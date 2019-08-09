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
		fmt.Println("Remote server just fetched our /actor endpoint")

		username := mux.Vars(r)["actor"]
		// TODO replace this with a LoadActor that loads an actor from the database with this username
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
