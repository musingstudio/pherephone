package main

import (
	"context"
	"io/ioutil"
	"net/url"
	"os"

	// "strings"

	"github.com/go-fed/activity/streams"
	"github.com/go-fed/activity/streams/vocab"

	"encoding/json"
	"log"

	"github.com/davecgh/go-spew/spew"
	"github.com/dchest/uniuri"
)

type database struct {
	grandparent *Actor
}

func (d *database) Lock(c context.Context, id *url.URL) error {
	//log.Println("db")
	return nil
}

func (d *database) Unlock(c context.Context, id *url.URL) error {
	//log.Println("db")
	return nil
}

func (d *database) NewId(c context.Context, t vocab.Type) (id *url.URL, err error) {

	log.Println("newID")

	unique := uniuri.New()
	id, err = url.Parse(baseURL + "post/" + unique)

	return
}

func (d *database) InboxContains(c context.Context, inbox, id *url.URL) (contains bool, err error) {
	//log.Println("db")
	return
}

func (d *database) GetInbox(c context.Context, inboxIRI *url.URL) (inbox vocab.ActivityStreamsOrderedCollectionPage, err error) {
	log.Println("getInboxdb")
	inbox = streams.NewActivityStreamsOrderedCollectionPage()
	return
}

func (d *database) SetInbox(c context.Context, inbox vocab.ActivityStreamsOrderedCollectionPage) error {
	//log.Println("db")
	var err error
	return err
}

func (d *database) Owns(c context.Context, id *url.URL) (owns bool, err error) {
	//log.Println("db")
	return
}

func (d *database) ActorForOutbox(c context.Context, outboxIRI *url.URL) (actorIRI *url.URL, err error) {
	//log.Println("db")
	return
}

func (d *database) ActorForInbox(c context.Context, inboxIRI *url.URL) (actorIRI *url.URL, err error) {
	//log.Println("db")
	return
}

func (d *database) OutboxForInbox(c context.Context, inboxIRI *url.URL) (outboxIRI *url.URL, err error) {
	//log.Println("db")
	return
}

func (d *database) Exists(c context.Context, id *url.URL) (exists bool, err error) {
	//log.Println("db")
	return
}

func (d *database) Get(c context.Context, id *url.URL) (value vocab.Type, err error) {
	//log.Println("db")
	b := []byte(`{"@context": "https://www.w3.org/ns/activitystreams",
					"type": "Person",
					"id": "https://floorb.qwazix.com/",
					"name": "Alyssa P. Hacker",
					"preferredUsername": "alyssa",
					"summary": "Lisp enthusiast hailing from MIT",
					"inbox": "https://floorb.qwazix.com/inbox/",
					"outbox": "https://floorb.qwazix.com/outbox/",
					"followers": "https://floorb.qwazix.com/followers/",
					"following": "https://floorb.qwazix.com/following/",
					"liked": "https://floorb.qwazix.com/liked/"}`)
	var jsonMap map[string]interface{}
	if err = json.Unmarshal(b, &jsonMap); err != nil {
		panic(err)
	}

	value, err = streams.ToType(c, jsonMap)

	if err != nil {
		log.Println("something is wrong with the conversion of JSON to vocab.Type")
		return
	}

	return
}

func (d *database) Create(c context.Context, asType vocab.Type) (err error) {
	log.Println("createdb")
	serialized, _ := asType.Serialize()
	// spew.Dump(serialized)
	json, _ := json.MarshalIndent(serialized, "", "\t")
	spew.Println("======================================")

	// get anything after last slash of activitystreams id
	// split in slashes
	// slice := strings.Split(asType.GetActivityStreamsId().Get().String(), "/")
	// get last thing of slice
	// id := slice[len(slice)-1]

	id := makeURLsaveable(asType.GetActivityStreamsId().Get().String())

	// log.Println("this is id v")
	// log.Println(id)

	err = ioutil.WriteFile("actors"+slash+d.grandparent.name+slash+id, json, 0644)
	if err != nil {
		log.Printf("Unable to write outbox JSON to file: %+v", err)
		return err
	}

	return
}

func (d *database) Update(c context.Context, asType vocab.Type) (err error) {
	//log.Println("db")
	return
}

func (d *database) Delete(c context.Context, id *url.URL) (err error) {
	//log.Println("db")
	return
}

func (d *database) GetOutbox(c context.Context, outboxIRI *url.URL) (outbox vocab.ActivityStreamsOrderedCollectionPage, err error) {
	log.Println("getOutbox")
	outbox = streams.NewActivityStreamsOrderedCollectionPage()
	jsonFile := "actors" + slash + d.grandparent.name + slash + "outbox.json"
	orderedItems := streams.NewActivityStreamsOrderedItemsProperty()
	outbox.SetActivityStreamsOrderedItems(orderedItems)
	outboxData, err := readJSON(jsonFile)
	if err != nil {
		log.Println("Couldn't load outbox, creating...")
		d.SetOutbox(c, outbox)
		// we handled the error so nil
		return outbox, nil
	}
	if outboxData["orderedItems"] != nil {
		// it seems that if orderedItems hold only one value it's not an
		// array but a string
		// {
		// 	"orderedItems": "http://floorb.qwazix.com/post/kiAuK2c3czrW0saw",
		// 	"type": "OrderedCollectionPage"
		// }

		if item, ok := outboxData["orderedItems"].(string); ok {
			url, _ := url.Parse(item)
			orderedItems.AppendIRI(url)
		} else {
			// In the other case where it is indeed an array
			// {
			// 	"orderedItems": [
			// 		"http://floorb.qwazix.com/post/4nEZQsps5I0R9CXd",
			// 		"http://floorb.qwazix.com/post/3RltwfH9bToQpuq2"
			// 	],
			// 	"type": "OrderedCollectionPage"
			// }

			for _, item := range outboxData["orderedItems"].([]interface{}) {
				url, _ := url.Parse(item.(string))
				orderedItems.AppendIRI(url)
			}
		}
	}
	return
}

func (d *database) SetOutbox(c context.Context, outbox vocab.ActivityStreamsOrderedCollectionPage) error {
	log.Println("db.SetOutbox")
	serialized, _ := outbox.Serialize()
	json, _ := json.MarshalIndent(serialized, "", "\t")
	// spew.Dump(string(json))

	log.Println("Creating outbox for " + d.grandparent.name)
	// the actor ought to exist otherwise something is really wrong
	_, err := os.Stat("actors" + slash + d.grandparent.name)
	if err != nil {
		log.Println("Can't access actor diretory, something is wrong")
		return err
	}

	err = ioutil.WriteFile("actors"+slash+d.grandparent.name+slash+"outbox.json", json, 0644)
	if err != nil {
		log.Printf("Unable to write outbox JSON to file: %+v", err)
		return err
	}

	return err
}

func (d *database) Followers(c context.Context, actorIRI *url.URL) (followers vocab.ActivityStreamsCollection, err error) {
	////log.Println("db")
	return
}

func (d *database) Following(c context.Context, actorIRI *url.URL) (followers vocab.ActivityStreamsCollection, err error) {
	//log.Println("db")
	return
}

func (d *database) Liked(c context.Context, actorIRI *url.URL) (followers vocab.ActivityStreamsCollection, err error) {
	//log.Println("db")
	return
}
