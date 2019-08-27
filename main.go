package main

import (
	"fmt"
	// "log"
	"flag"
	"os"
	"strings"
	"strconv"

	// "errors"

	"encoding/json"
	"io/ioutil"
	"net/http"

	// "net/url"
	// "context"
	// "html"

	"github.com/go-fed/activity/pub"
	"github.com/gologme/log"

	// "github.com/go-fed/activity/streams"
	"github.com/gorilla/mux"
	"gopkg.in/ini.v1"
	// "github.com/davecgh/go-spew/spew"
)

var baseURL = "http://example.com/"
var storage = "storage"

func main() {

	var err error

	// This is here for debugging purposes. I want to be able to easily spot in the terminal
	// when a single execution starts
	fmt.Println()
	fmt.Println("======================= PHeRePHoNe ==========================")

	// introduce ourselves
	fmt.Println()
	fmt.Println("Pherephone follows some accounts and boosts")
	fmt.Println("whatever they post to our followers. See config.ini ")
	fmt.Println("for more information and how to set up. ")

	debugFlag := flag.Bool("debug", false, "set to true to get debugging information in the console")
	flag.Parse()

	// I prefer long file so that I can click it in the terminal and open it
	// in the editor above
	log.SetFlags(log.Llongfile)
	// log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.EnableLevel("warn")
	// create a logger with levels but without prefixes for easier to read
	// debug output
	printer := log.New(os.Stdout, " ", 0)

	if *debugFlag == true {
		fmt.Println()
		fmt.Println("debug mode on")
		log.EnableLevel("info")
		printer.EnableLevel("info")
	}

	// read configuration file (config.ini)
	cfg, err := ini.Load("config.ini")
	if err != nil {
		fmt.Printf("Fail to read file: %v", err)
		os.Exit(1)
	}

	// Load base url from configuration file
	baseURL = cfg.Section("general").Key("baseURL").String()
	// check if it ends with a / and append one if not
	if baseURL[len(baseURL)-1:] != "/" {
		baseURL += "/"
	}
	// print it for our users
	fmt.Println()
	fmt.Println("Domain Name:", baseURL)

	// Load storage location (only local filesystem supported for now) from config
	storage = cfg.Section("general").Key("storage").String()
	cwd, err := os.Getwd()
	fmt.Println("Storage Location:", cwd+slash+storage)
	fmt.Println()

	// prepare storage for foreign activities (activities we store that don't
	// belong to us)
	foreignDir := storage + slash + "foreign"
	if _, err := os.Stat(foreignDir); os.IsNotExist(err) {
		os.MkdirAll(foreignDir, 0755)
	}

	// This could work too if we don't want to handle multiple actors and stuff:
	// var outboxHandler http.HandlerFunc = actor.HandleOutbox
	// might consider moving the whole thing to another file
	var outboxHandler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/activity+json; charset=utf-8")
		username := mux.Vars(r)["actor"]  // get the needed actor from the muxer (url variable {actor} below)
		actor, err := LoadActor(username) // load the actor from disk
		if err != nil {                   // either actor requested has illegal characters or
			log.Info("Can't load local actor") // we don't have such actor
			fmt.Fprintf(w, "404 - page not found")
			w.WriteHeader(http.StatusNotFound)
			return
		}
		// check if the ActivityPub headers exist
		// Content-Type: application/activity+json; profile="https://www.w3.org/ns/activitystreams"
		// Accept: application/activity+json
		// pherephone doesn't have a web interface though so
		// maybe I shouldn't make debugging harder (TBD) and let
		// the user see the json in the browser
		if pub.IsActivityPubRequest(r) {
			actor.HandleOutbox(w, r)
		} else {
			// The above does nothing if it's a non-ActivityPub request so
			// handle non-ActivityPub request here, such as serving a webpage.
		}
	}
	var inboxHandler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		log.Info(r.Header)
		w.Header().Set("content-type", "application/activity+json; charset=utf-8")
		// Populate c with request-specific information
		username := mux.Vars(r)["actor"]
		// Load actor information from storage/actors/<actor>/<actor>.json
		// (not literal actor.json) (that one contains the activitystreams
		// implementation of actor, that doesn't take care of storing followers)
		actor, err := LoadActor(username)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			log.Info("Can't load local actor")
			fmt.Fprintf(w, "404 - page not found")
			return
		}
		if pub.IsActivityPubRequest(r) {
			actor.HandleInbox(w, r)
		} else {
			// The above does nothing if it's a non-ActivityPub request so
			// handle non-ActivityPub request here, such as serving a webpage.
		}
	}

	var actorHandler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/activity+json; charset=utf-8")
		log.Info("Remote server just fetched our /actor endpoint")
		username := mux.Vars(r)["actor"]
		log.Info(username)
		if username == ".well-known" || username == "favicon.ico" {
			log.Info("well-known, skipping...")
			return
		}
		actor, err := LoadActor(username)
		// error out if this actor does not exist (or there are dots or slashes in his name)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "404 - page not found")
			log.Info("Can't create local actor")
			return
		}
		fmt.Fprintf(w, actor.whoAmI())
		log.Info(r.RemoteAddr)
		log.Info(r.Body)
		log.Info(r.Header)
		// log.Info(actor.whoAmI())
	}

	var postHandler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/activity+json; charset=utf-8")
		username := mux.Vars(r)["actor"]
		hash := mux.Vars(r)["hash"]
		actor, err := LoadActor(username)
		// error out if this actor does not exist
		if err != nil {
			log.Info("Can't create local actor")
			return
		}
		post, err := actor.getPost(hash)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "404 - post not found")
			return
		}
		fmt.Fprintf(w, post)
	}

	var followersHandler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/activity+json; charset=utf-8")
		username := mux.Vars(r)["actor"]
		actor, err := LoadActor(username)
		// error out if this actor does not exist
		if err != nil {
			log.Info("Can't create local actor")
			return
		}
		var page int
		pageS := r.URL.Query().Get("page")
		if pageS == "" {
			page = 0
		} else {
			page, _ = strconv.Atoi(pageS)
		}
		response, _ := actor.GetFollowers(page)
		w.Write(response)
	}

	var followingHandler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/activity+json; charset=utf-8")
		username := mux.Vars(r)["actor"]
		actor, err := LoadActor(username)
		// error out if this actor does not exist
		if err != nil {
			log.Info("Can't create local actor")
			return
		}
		var page int
		pageS := r.URL.Query().Get("page")
		if pageS == "" {
			page = 0
		} else {
			page, _ = strconv.Atoi(pageS)
		}
		response, _ := actor.GetFollowing(page)
		w.Write(response)
	}

	var webfingerHandler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/jrd+json; charset=utf-8")
		account := r.URL.Query().Get("resource")              // should be something like acct:user@example.com
		account = strings.Replace(account, "acct:", "", 1)    // remove acct:
		server := strings.Split(baseURL, "://")[1]            // remove protocol from baseURL. Should get example.com
		server = strings.TrimSuffix(server, "/")              // remove protocol from baseURL. Should get example.com
		account = strings.Replace(account, "@"+server, "", 1) // remove server from handle. Should get user
		actor, err := LoadActor(account)
		// error out if this actor does not exist
		if err != nil {
			log.Info("No such actor")
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "404 - post not found")
			return
		}
		// response := `{"subject":"acct:` + actor.name + `@` + server + `","aliases":["` + baseURL + actor.name + `","` + baseURL + actor.name + `"],"links":[{"href":"` + baseURL + `","type":"text/html","rel":"https://webfinger.net/rel/profile-page"},{"href":"` + baseURL + actor.name + `","type":"application/activity+json","rel":"self"}]}`

		response := `{
			"subject": "acct:` + actor.name + `@` + server + `",
			"links": [
				{
					"rel": "self",
					"type": "application/activity+json",
					"href": "` + baseURL + actor.name + `"
				}
			]
		}`
		log.Info(response)
		w.Write([]byte(response))
	}

	// Add the handlers to a HTTP server
	gorilla := mux.NewRouter()
	gorilla.HandleFunc("/.well-known/webfinger", webfingerHandler)
	gorilla.HandleFunc("/{actor}/outbox", outboxHandler)
	gorilla.HandleFunc("/{actor}/outbox/", outboxHandler)
	gorilla.HandleFunc("/{actor}/inbox", inboxHandler)
	gorilla.HandleFunc("/{actor}/inbox/", inboxHandler)
	gorilla.HandleFunc("/{actor}/followers", followersHandler)
	gorilla.HandleFunc("/{actor}/followers/", followersHandler)
	gorilla.HandleFunc("/{actor}/following", followingHandler)
	gorilla.HandleFunc("/{actor}/following/", followingHandler)
	gorilla.HandleFunc("/{actor}", actorHandler)
	gorilla.HandleFunc("/{actor}/", actorHandler)
	gorilla.HandleFunc("/{actor}/post/{hash}", postHandler)
	http.Handle("/", gorilla)

	// Here we begin the actual pherephone functionality

	// get the list of local actors to host and the list
	// of remote actors any of them relays (boosts, announces)
	jsonFile, err := os.Open("actors.json")
	if err != nil {
		log.Info("something is wrong with the json file containing the actors")
		log.Info(err)
	}

	// Unmarshall it into a map of string arrays
	whoFollowsWho := make(map[string]map[string]interface{})
	byteValue, _ := ioutil.ReadAll(jsonFile)
	json.Unmarshal(byteValue, &whoFollowsWho)

	// log.Info(string(byteValue))
	// create all local actors if they don't exist yet
	for follower, data := range whoFollowsWho {
		followees := data["follow"].([]interface{})
		printer.Info()
		log.Info("Local Actor: " + follower)
		if strings.ContainsAny(follower, " \\/:*?\"<>|") {
			log.Warn("local actors can't have spaces or any of these characters in their name: \\/:*?\"<>|")
			log.Warn("Actor " + follower + " will be ignored")
			continue
		}
		followerActor, err := GetActor(follower, data["summary"].(string), "Service", baseURL+follower)
		if err != nil {
			log.Info("error creating local follower")
			return
		}
		// Now follow each one of it's users
		log.Info("Users to relay:")
		for _, followee := range followees {
			log.Info(followee)
			followerActor.Follow(followee.(string))
		}
		
	}
	log.Fatal(http.ListenAndServe(":8081", nil))
}
