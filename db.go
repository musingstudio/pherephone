package main

import (
	"context"
	"io/ioutil"
	"net/url"
	"os"

	"strings"
	"unicode/utf8"

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
	id, err = url.Parse(baseURL + d.grandparent.name + "/" + unique)

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
	stringURL := id.String()
	return strings.HasPrefix(stringURL, baseURL), nil
}

func (d *database) ActorForOutbox(c context.Context, outboxIRI *url.URL) (actorIRI *url.URL, err error) {
	stringURL := outboxIRI.String()
	actor := strings.Replace(strings.Replace(stringURL, "/outbox", "", -1), baseURL, "", -1)
	actorIRI, _ = url.Parse(actor)
	return
}

func (d *database) ActorForInbox(c context.Context, inboxIRI *url.URL) (actorIRI *url.URL, err error) {
	stringURL := inboxIRI.String()
	actor := strings.Replace(strings.Replace(stringURL, "/inbox", "", -1), baseURL, "", -1)
	actorIRI, _ = url.Parse(actor)
	return
}

func (d *database) OutboxForInbox(c context.Context, inboxIRI *url.URL) (outboxIRI *url.URL, err error) {
	stringURL := inboxIRI.String()
	outboxURL := strings.Replace(stringURL, "inbox", "outbox", -1)
	outboxIRI, _ = url.Parse(outboxURL)
	return
}

func (d *database) Exists(c context.Context, id *url.URL) (exists bool, err error) {
	var jsonFile string
	if owns, _ := d.Owns(c, id); owns {
		actor, hash := d.parseIRI(id)
		jsonFile = storage + slash + "actors" + slash + actor + slash + hash + ".json"
		// this should look like storage/actors/qwazix/nvjfdjelkjdjk.json
		// or storage/actors/qwazix/qwazix.json
	} else {
		path := makeURLsaveable(strings.Replace(id.String(), baseURL, "", 1))
		jsonFile = storage + slash + "foreign" + slash + path + ".json"
		// this should look like storage/foreign/http:😆😆some.domain😆some😆path.json
	}
	_, err = os.Stat(jsonFile)

	return err == nil, err
}

func (d *database) parseIRI(id *url.URL) (actor string, hash string) {
	idString := id.String()
	// check if the last character is a slash and if it is, remove it
	if last, size := utf8.DecodeLastRuneInString(idString); last == rune('/') {
		idString = idString[:len(idString)-size]
	}
	// remove the baseURL
	idString = strings.Replace(idString, baseURL, "", 1)
	// split with slashes
	slice := strings.Split(idString, "/")
	// first part is always the actor
	actor = slice[0]
	// if we only have an actor the json is named after the actor
	hash = "actor"
	// if the slice has other things then we have an activity
	if len(slice) > 1 {
		// get last thing of slice (random string part of id)
		hash = slice[1]
	}
	return
}

func (d *database) Get(c context.Context, id *url.URL) (value vocab.Type, err error) {
	log.Println("call to Get with id: " + id.String())
	//TODO: replace / with \ for windows
	var jsonFile string
	if owns, _ := d.Owns(c, id); owns {
		actor, hash := d.parseIRI(id)
		jsonFile = storage + slash + "actors" + slash + actor + slash + hash + ".json"
		// this should look like storage/actors/qwazix/nvjfdjelkjdjk.json
		// or storage/actors/qwazix/qwazix.json
	} else {
		path := makeURLsaveable(strings.Replace(id.String(), baseURL, "", 1))
		jsonFile = storage + slash + "foreign" + slash + path + ".json"
		// this should look like storage/foreign/http:😆😆some.domain😆some😆path.json
	}
	jsonMap, err := readJSON(jsonFile)
	if err != nil {
		log.Println("probably the item doesn't exist in our database")
		return
	}
	spew.Dump(jsonMap)
	value, err = streams.ToType(c, jsonMap)
	if err != nil {
		log.Println("something is wrong with the conversion of JSON to vocab.Type")
		return
	}
	return
}

func (d *database) Create(c context.Context, asType vocab.Type) (err error) {
	log.Println("call to create with " + asType.GetActivityStreamsId().GetIRI().String())
	serialized, _ := asType.Serialize()
	// spew.Dump(serialized)
	json, _ := json.MarshalIndent(serialized, "", "\t")
	spew.Println("======================================")

	// get anything after last slash of activitystreams id
	// split in slashes
	// slice := strings.Split(asType.GetActivityStreamsId().Get().String(), "/")
	// get last thing of slice
	// id := slice[len(slice)-1]

	id := asType.GetActivityStreamsId().Get()

	// if any of our actors own this
	var actor, hash, filename string
	if owns, _ := d.Owns(c, id); owns {
		// split the url and put it in the respective actor
		actor, hash = d.parseIRI(id)
		filename = storage + slash + "actors" + slash + actor + slash + hash + ".json"
	} else {
		// create a `foreign` directory, replace slashes with smileys and put it there
		filepart := makeURLsaveable(asType.GetActivityStreamsId().Get().String())
		filename = storage + slash + "foreign" + slash + filepart + ".json"
	}
	// log.Println("this is id v")
	// log.Println(id)

	err = ioutil.WriteFile(filename, json, 0644)
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

// GetOutbox actually unserializes the outbox json back to vocab.ActivityStreamsOrderedCollectionPage
func (d *database) GetOutbox(c context.Context, outboxIRI *url.URL) (outbox vocab.ActivityStreamsOrderedCollectionPage, err error) {
	log.Println("getOutbox")
	outbox = streams.NewActivityStreamsOrderedCollectionPage()
	jsonFile := storage + slash + "actors" + slash + d.grandparent.name + slash + "outbox.json"
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

// SetOutbox is being fed with the new outbox and we have to compare it with the old outbox and implement the differences. In our
// case we just overwrite it because we don't have any structured data.
func (d *database) SetOutbox(c context.Context, outbox vocab.ActivityStreamsOrderedCollectionPage) error {
	log.Println("db.SetOutbox")
	serialized, _ := outbox.Serialize()
	json, _ := json.MarshalIndent(serialized, "", "\t")
	// spew.Dump(string(json))

	log.Println("Creating outbox for " + d.grandparent.name)
	// the actor ought to exist otherwise something is really wrong
	_, err := os.Stat(storage + slash + "actors" + slash + d.grandparent.name)
	if err != nil {
		log.Println("Can't access actor diretory, something is wrong")
		return err
	}

	err = ioutil.WriteFile(storage+slash+"actors"+slash+d.grandparent.name+slash+"outbox.json", json, 0644)
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
