package main

import (
	"fmt"

	"github.com/go-fed/activity/streams"

	// "github.com/go-fed/activity/streams"
	"github.com/go-fed/activity/pub"
	// "errors"
	"log"
	"net/http"
	"net/url"

	"encoding/json"
	"io/ioutil"
	"os"

	// "html"
	"context"
)

func main() {

	fmt.Println("=========================================================================")

	var clock *clock
	var err error
	var db *database

	clock, err = newClock("Europe/Athens")
	if err != nil {
		return
	}

	common := newCommonBehavior(db)
	federating := newFederatingBehavior(db)
	actor := pub.NewFederatingActor(common, federating, db, clock)

	var outboxHandler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		c := context.Background()
		// Populate c with request-specific information
		if handled, err := actor.PostOutbox(c, w, r); err != nil {
			// Write to w
			return
		} else if handled {
			return
		} else if handled, err = actor.GetOutbox(c, w, r); err != nil {
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
		if handled, err := actor.PostInbox(c, w, r); err != nil {
			// Write to w
			return
		} else if handled {
			return
		} else if handled, err = actor.GetInbox(c, w, r); err != nil {
			// Write to w
			return
		} else if handled {
			return
		}

		// else:
		//
		// Handle non-ActivityPub request, such as serving a webpage.
	}
	// Add the handlers to a HTTP server
	//   serveMux := http.NewServeMux()
	http.HandleFunc("/actor/outbox", outboxHandler)
	http.HandleFunc("/actor/inbox", inboxHandler)

	// get the list of users to relay
	jsonFile, err := os.Open("actors.json")

	if err != nil {
		fmt.Println("something is wrong with the json file containing the actors")
		fmt.Println(err)
	}

	var actors []string

	byteValue, _ := ioutil.ReadAll(jsonFile)
	json.Unmarshal(byteValue, &actors)

	// Now for each one of these users get their outbox
	fmt.Println("Users to relay:")
	for _, user := range actors {
		fmt.Println(user)
		// ra := NewRemoteActor(user)
		// fmt.Println(ra.outboxIri)

		c := context.Background()

		follow := streams.NewActivityStreamsFollow()
		object := streams.NewActivityStreamsObjectProperty()
		to := streams.NewActivityStreamsToProperty()
		iri, err := url.Parse(user)
		// iri, err := url.Parse("https://print3d.social/users/qwazix/outbox")
		if err != nil {
			fmt.Println("something is wrong when parsing the remote" +
				"actors iri into a url")
			fmt.Println(err)
			return
		}
		to.AppendIRI(iri)
		object.AppendIRI(iri)
		follow.SetActivityStreamsObject(object)
		follow.SetActivityStreamsTo(to)

		iri, err = url.Parse("http://floorb.qwazix.com/actor/outbox")

		if err != nil {
			fmt.Println("something is wrong when parsing the local" +
				"actors iri into a url")
			fmt.Println(err)
			return
		}

		// fmt.Println(c)
		// fmt.Println(iri)
		// fmt.Println(follow)

		actor.Send(c, iri, follow)
		// PrettyPrint(ra.getLatestPosts(10))
	}

	http.HandleFunc("/hi", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hi")
	})

	log.Fatal(http.ListenAndServe(":8081", nil))

}
