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

	// "html"
	// "context"

	// "github.com/davecgh/go-spew/spew"
)

var domainName = "http://floorb.qwazix.com"

func main() {

	var err error

	fmt.Println("=========================================================================")

	var outboxHandler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		username := mux.Vars(r)["actor"]
		// TODO replace this with a LoadActor that loads an actor from the database with this username
		actor, err := MakeActor(username, "My name is"+username, "Service", domainName+"/"+username)
		if err != nil {
			fmt.Println("Can't create local actor")
			return
		}
		if pub.IsActivityPubRequest(r){
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
		actor, err := MakeActor(username, "My name is"+username, "Service", domainName+"/"+username)
		if err != nil {
			fmt.Println("Can't create local actor")
			return
		}
		if pub.IsActivityPubRequest(r){
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
		actor, err := MakeActor(username, "My name is"+username, "Service", domainName+"/"+username)
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

	// get the list of users to relay
	jsonFile, err := os.Open("actors.json")

	if err != nil {
		fmt.Println("something is wrong with the json file containing the actors")
		fmt.Println(err)
	}

	// Unmarshall it into a map of string arrays
	whoFollowsWho := make(map[string][]string)
	byteValue, _ := ioutil.ReadAll(jsonFile)
	json.Unmarshal(byteValue, &whoFollowsWho)

	// fmt.Println(string(byteValue))
	
	for follower, followees := range whoFollowsWho {
		// Now follow each one of these users
		// I want to focus on handling the incoming messages before
		// dealing with databases so I will just comment this out for now
		fmt.Println("Local Actor: "+ follower)
		followerActor, err := MakeActor(follower, "emptySummary", "Service", domainName+"/"+follower)
		if err != nil {
			fmt.Println("error creating local follower")
			return
		}
		fmt.Println("Users to relay:")
		for _, followee := range followees {
			fmt.Println(followee)
			if err != nil{
				fmt.Println("Couldn't create local actor")
				return
			}
			followerActor.Follow(followee)
		}
	}

	http.HandleFunc("/hi", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hi")
	})

	log.Fatal(http.ListenAndServe(":8081", nil))

}
