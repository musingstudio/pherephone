package main

import (
	"fmt"

	// "github.com/go-fed/activity/streams"
	"github.com/go-fed/activity/pub"
	"github.com/gorilla/mux"

	// "errors"
	"log"
	"net/http"

	// "net/url"

	"encoding/json"
	"io/ioutil"
	"os"

	"gopkg.in/ini.v1"
	// "html"
	// "context"
	// "github.com/davecgh/go-spew/spew"
)

var baseURL = "http://example.com"

func main() {

	var err error

	// This is here for debugging purposes. I want to be able to easily spot in the terminal
	// when a single execution starts
	fmt.Println("=========================================================================")

	// read configuration file (config.ini)
	cfg, err := ini.Load("config.ini")
	if err != nil {
		fmt.Printf("Fail to read file: %v", err)
		os.Exit(1)
	}
	// config.ini for now only contains the baseURL
	// Load base url from configuration file
	baseURL = cfg.Section("general").Key("baseURL").String()
	fmt.Println("Domain Name:", baseURL)

	// This could work too if we don't want to handle multiple actors and stuff
	// var outboxHandler http.HandlerFunc = actor.HandleOutbox
	// might consider moving the whole thing to another file
	var outboxHandler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		username := mux.Vars(r)["actor"]
		actor, err := LoadActor(username)
		if err != nil {
			fmt.Println("Can't create local actor")
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
		// TODO replace this with a LoadActor that loads an actor from the database with this username
		actor, err := LoadActor(username)
		if err != nil {
			fmt.Println("Can't create local actor")
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
		fmt.Println("Remote server just fetched our /actor endpoint")

		username := mux.Vars(r)["actor"]
		// TODO replace this with a LoadActor that loads an actor from the database with this username
		// error out if this actor does not exist
		actor, err := LoadActor(username)
		if err != nil {
			fmt.Println("Can't create local actor")
			return
		}
		fmt.Fprintf(w, actor.whoAmI())
	}

	// Add the handlers to a HTTP server
	gorilla := mux.NewRouter()
	gorilla.HandleFunc("/{actor}/outbox", outboxHandler)
	gorilla.HandleFunc("/{actor}/inbox", inboxHandler)
	gorilla.HandleFunc("/{actor}/inbox/", inboxHandler)
	gorilla.HandleFunc("/{actor}", actorHandler)
	gorilla.HandleFunc("/{actor}/", actorHandler)
	http.Handle("/", gorilla)

	// Here we begin the actual pherephone functionality

	// get the list of local actors to host and the list
	// of remote actors any of them relays (boosts, announces)
	jsonFile, err := os.Open("actors.json")
	if err != nil {
		fmt.Println("something is wrong with the json file containing the actors")
		fmt.Println(err)
	}

	// Unmarshall it into a map of string arrays
	// TODO add summary thus making this map[string]interface{}
	whoFollowsWho := make(map[string][]string)
	byteValue, _ := ioutil.ReadAll(jsonFile)
	json.Unmarshal(byteValue, &whoFollowsWho)

	// fmt.Println(string(byteValue))
	// create all local actors if they don't exist yet
	for follower, followees := range whoFollowsWho {
		fmt.Println()
		fmt.Println("Local Actor: " + follower)
		followerActor, err := GetActor(follower, "emptySummary", "Service", baseURL+"/"+follower)
		if err != nil {
			fmt.Println("error creating local follower")
			return
		}
		// Now follow each one of it's users
		fmt.Println("Users to relay:")
		for _, followee := range followees {
			fmt.Println(followee)
			followerActor.Follow(followee)
		}
	}
	log.Fatal(http.ListenAndServe(":8081", nil))
}
