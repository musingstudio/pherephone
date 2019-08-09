package main

import (
	"fmt"

	// "github.com/go-fed/activity/streams"
	"github.com/go-fed/activity/pub"
	// "errors"
	"log"
	"net/http"

	// "net/url"

	"encoding/json"
	"io/ioutil"
	"os"

	// "html"
	"context"
)

var domainName string = "http://floorb.qwazix.com"

func main() {

	var clock *clock
	var err error
	var db *database

	fmt.Println("=========================================================================")

	clock, err = newClock("Europe/Athens")
	if err != nil {
		return
	}

	common := newCommonBehavior(db)
	federating := newFederatingBehavior(db)
	pubActor := pub.NewFederatingActor(common, federating, db, clock)

	var outboxHandler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		c := context.Background()
		// Populate c with request-specific information
		if handled, err := pubActor.PostOutbox(c, w, r); err != nil {
			// Write to w
			return
		} else if handled {
			return
		} else if handled, err = pubActor.GetOutbox(c, w, r); err != nil {
			// Write to w
			return
		} else if handled {
			fmt.Println("gethandled")
			return
		}
		// else:
		//
		// Handle non-ActivityPub request, such as serving a webpage.
	}
	var inboxHandler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		c := context.Background()
		// Populate c with request-specific information
		if handled, err := pubActor.PostInbox(c, w, r); err != nil {
			fmt.Println(err)
			// Write to w
			return
		} else if handled {
			return
		} else if handled, err = pubActor.GetInbox(c, w, r); err != nil {
			// Write to w
			return
		} else if handled {
			return
		}

		// else:
		//
		// Handle non-ActivityPub request, such as serving a webpage.
	}

	var actorHandler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		// Populate c with request-specific information
		fmt.Fprintf(w, `{"@context":	"https://www.w3.org/ns/activitystreams",
										"type": "Person",
										"id": "http://floorb.qwazix.com/actor/",
										"name": "Alyssa P. Hacker",
										"preferredUsername": "pherephone",
										"summary": "pherephone is somebody that repeats others",
										"inbox": "http://floorb.qwazix.com/actor/inbox/",
										"outbox": "http://floorb.qwazix.com/actor/outbox/",
										"followers": "http://floorb.qwazix.com/actor/followers/",
										"following": "http://floorb.qwazix.com/actor/following/",
										"liked": "http://floorb.qwazix.com/actor/liked/"}`)
		fmt.Println("Remote server just fetched our /actor endpoint")
	}

	// Add the handlers to a HTTP server
	//   serveMux := http.NewServeMux()
	http.HandleFunc("/actor/outbox", outboxHandler)
	http.HandleFunc("/actor/inbox", inboxHandler)
	http.HandleFunc("/actor/inbox/", inboxHandler)
	http.HandleFunc("/actor", actorHandler)
	http.HandleFunc("/actor/", actorHandler)

	// get the list of users to relay
	jsonFile, err := os.Open("actors.json")

	if err != nil {
		fmt.Println("something is wrong with the json file containing the actors")
		fmt.Println(err)
	}

	var actors []string

	byteValue, _ := ioutil.ReadAll(jsonFile)
	json.Unmarshal(byteValue, &actors)

	// Now follow each one of these users
	// I want to focus on handling the incoming messages before
	// dealing with databases so I will just comment this out for now
	// fmt.Println("Users to relay:")
	// for _, user := range actors {
	// 	fmt.Println(user)
	// 	actor, err := MakeActor("Pherephone", "pherephone repeats", "service", "http://floorb.qwazix.com/actor")
	// 	if err != nil{
	// 		fmt.Println("Couldn't create local actor")
	// 		return
	// 	}
	// 	actor.Follow(user)
	// }

	http.HandleFunc("/hi", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hi")
	})

	log.Fatal(http.ListenAndServe(":8081", nil))

}
