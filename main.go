package main

import (
	"os"
	"fmt"
	"log"
	// "strings"
	// "errors"

	"encoding/json"
	"io/ioutil"
	"net/http"
	// "net/url"
	// "context"
	// "html"

	"github.com/go-fed/activity/pub"
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
	log.Println("======================= PHeRePHoNe ==========================")

	// I prefer long file so that I can click it in the terminal and open it
	// in the editor above
	log.SetFlags(log.Llongfile)
	// log.SetFlags(log.LstdFlags | log.Lshortfile)

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
	log.Println("Domain Name:", baseURL)

	// Load storage location (only local filesystem supported for now) from config
	storage = cfg.Section("general").Key("storage").String()
	log.Println("Storage Location:", storage)
	log.Println()
	log.Println("Take a look at config.ini if you want to change the above values")

	// prepare storage
	if _, err := os.Stat(storage + slash + "foreign"); os.IsNotExist(err) {
		os.MkdirAll(storage+slash+"foreign", 0755)
	}

	// This could work too if we don't want to handle multiple actors and stuff:
	// var outboxHandler http.HandlerFunc = actor.HandleOutbox
	// might consider moving the whole thing to another file
	var outboxHandler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		username := mux.Vars(r)["actor"]
		actor, err := LoadActor(username)
		if err != nil {
			log.Println("Can't create local actor")
			fmt.Fprintf(w, "404 - page not found")
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if pub.IsActivityPubRequest(r) {
			actor.HandleOutbox(w, r)
		} else {
			// The above does nothing if it's a non-ActivityPub request so
			// handle non-ActivityPub request here, such as serving a webpage.
		}
	}
	var inboxHandler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		// Populate c with request-specific information
		username := mux.Vars(r)["actor"]
		// Load actor information from storage/actors/<actor>/<actor>.json
		// (not literal actor.json) (that one contains the activitystreams
		// implementation of actor, that doesn't take care of storing followers)
		actor, err := LoadActor(username)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			log.Println("Can't create local actor")
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
		log.Println("Remote server just fetched our /actor endpoint")

		username := mux.Vars(r)["actor"]
		actor, err := LoadActor(username)
		// error out if this actor does not exist (or there are dots or slashes in his name)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "404 - page not found")
			log.Println("Can't create local actor")
			return
		}
		fmt.Fprintf(w, actor.whoAmI())
	}

	var postHandler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		username := mux.Vars(r)["actor"]
		hash := mux.Vars(r)["hash"]
		actor, err := LoadActor(username)
		// error out if this actor does not exist
		if err != nil {
			log.Println("Can't create local actor")
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

	// Add the handlers to a HTTP server
	gorilla := mux.NewRouter()
	gorilla.HandleFunc("/{actor}/outbox", outboxHandler)
	gorilla.HandleFunc("/{actor}/inbox", inboxHandler)
	gorilla.HandleFunc("/{actor}/inbox/", inboxHandler)
	gorilla.HandleFunc("/{actor}", actorHandler)
	gorilla.HandleFunc("/{actor}/", actorHandler)
	gorilla.HandleFunc("/{actor}/post/{hash}", postHandler)
	http.Handle("/", gorilla)

	// Here we begin the actual pherephone functionality

	// get the list of local actors to host and the list
	// of remote actors any of them relays (boosts, announces)
	jsonFile, err := os.Open("actors.json")
	if err != nil {
		log.Println("something is wrong with the json file containing the actors")
		log.Println(err)
	}

	// Unmarshall it into a map of string arrays
	// TODO add summary thus making this map[string]interface{}
	whoFollowsWho := make(map[string]map[string]interface{})
	byteValue, _ := ioutil.ReadAll(jsonFile)
	json.Unmarshal(byteValue, &whoFollowsWho)

	// log.Println(string(byteValue))
	// create all local actors if they don't exist yet
	for follower, data := range whoFollowsWho {
		log.Println(data["follow"])
		followees := data["follow"].([]interface{})
		log.Println()
		log.Println("Local Actor: " + follower)
		followerActor, err := GetActor(follower, data["summary"].(string), "Service", baseURL+follower)
		if err != nil {
			log.Println("error creating local follower")
			return
		}
		// Now follow each one of it's users
		log.Println("Users to relay:")
		for _, followee := range followees {
			log.Println(followee)
			followerActor.Follow(followee.(string))
		}
	}
	log.Fatal(http.ListenAndServe(":8081", nil))
}
