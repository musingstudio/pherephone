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
	// "log"
	"github.com/gologme/log"

	// "github.com/davecgh/go-spew/spew"
	"github.com/dchest/uniuri"
)

type database struct {
	grandparent *Actor
}

// Lock does nothing. All interactions except being followed happen on separate files so
// it is probably an overkill to create lock files. Will investigate.
func (d *database) Lock(c context.Context, id *url.URL) error {
	//log.Info("db")
	return nil
}

// Unlock does nothing. All interactions except being followed happen on separate files so
// it is probably an overkill to create lock files. Will investigate.
func (d *database) Unlock(c context.Context, id *url.URL) error {
	//log.Info("db")
	return nil
}

// NewId produces a long enough alphanumeric hash so that collisions should be negligible
func (d *database) NewId(c context.Context, t vocab.Type) (id *url.URL, err error) {
	log.Info("newID")
	// uniuri is a random uri generation library
	unique := uniuri.New()
	id, err = url.Parse(baseURL + d.grandparent.name + "/" + unique)
	return
}

// InboxContains returns always false as we don't actually store anything in our inbox
func (d *database) InboxContains(c context.Context, inbox, id *url.URL) (contains bool, err error) {
	// When we receive a post we just announce it
	// I will see if this is required in any way
	return
}

// GetInbox returns an empty ordered collection page. It's here just to satisfy the interface.
func (d *database) GetInbox(c context.Context, inboxIRI *url.URL) (inbox vocab.ActivityStreamsOrderedCollectionPage, err error) {
	log.Info("getInboxdb")
	// we always return an empty list (see comment above)
	inbox = streams.NewActivityStreamsOrderedCollectionPage()
	return
}

// SetInbox does nothing. It's here just to satisfy the interface.
func (d *database) SetInbox(c context.Context, inbox vocab.ActivityStreamsOrderedCollectionPage) error {
	// as above
	var err error
	return err
}

// Owns checks if an activity originates from one of our local actors or some external one (on another server)
func (d *database) Owns(c context.Context, id *url.URL) (owns bool, err error) {
	stringURL := id.String()
	return strings.HasPrefix(stringURL, baseURL), nil
}

// ActorForOutbox returns the IRI of the actor that owns an outbox
// It just strips "/outbox" from the end of the string
func (d *database) ActorForOutbox(c context.Context, outboxIRI *url.URL) (actorIRI *url.URL, err error) {
	stringURL := outboxIRI.String()
	actor := strings.Replace(stringURL, "/outbox", "", -1)
	actorIRI, _ = url.Parse(actor)
	return
}

// ActorForOutbox returns the IRI of the actor that owns an inbox
// It just strips "/inbox" from the end of the string
func (d *database) ActorForInbox(c context.Context, inboxIRI *url.URL) (actorIRI *url.URL, err error) {
	stringURL := inboxIRI.String()
	actor := strings.Replace(stringURL, "/inbox", "", -1)
	actorIRI, _ = url.Parse(actor)
	return
}

// ActorForOutbox returns the IRI of the outbox of the actor that owns an inbox
// takes http://example.com/actor/inbox and makes it to http://example.com/actor/outbox
func (d *database) OutboxForInbox(c context.Context, inboxIRI *url.URL) (outboxIRI *url.URL, err error) {
	stringURL := inboxIRI.String()
	outboxURL := strings.Replace(stringURL, "inbox", "outbox", -1)
	outboxIRI, _ = url.Parse(outboxURL)
	return
}

// Exists checks whether we have an activity stored regardless if it has originated
// from one of our own actors (`Owns()`)
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
	return err == nil, nil
}

// parseIRI splits a post IRI and returs the actor that posted it and
// the hash that identifies it
func (d *database) parseIRI(id *url.URL) (actor string, hash string) {
	idString := id.String()
	// check if the last character is a slash and if it is, remove it
	// these are always / and not `slash` because they are URI's and not
	// local filesystem separators
	if last, size := utf8.DecodeLastRuneInString(idString); last == rune('/') {
		idString = idString[:len(idString)-size]
	}
	// remove the baseURL
	idString = strings.Replace(idString, baseURL, "", 1)
	// split with slashes
	slice := strings.Split(idString, "/")
	// first part is always the actor
	actor = slice[0]
	// if we only have an actor the json is named actor.json
	hash = "actor"
	// if the slice has other things then we have an activity
	if len(slice) > 1 {
		// get last thing of slice (random string part of id)
		hash = slice[1]
	}
	return
}

// Get fetches a post from our storage by IRI
func (d *database) Get(c context.Context, id *url.URL) (value vocab.Type, err error) {
	log.Info("call to Get with id: " + id.String())
	var jsonFile string
	// I don't know if there's a point in handling all the errors here, too much verbosity
	// TBD
	if owns, _ := d.Owns(c, id); owns {
		actor, hash := d.parseIRI(id)
		jsonFile = storage + slash + "actors" + slash + actor + slash + hash + ".json"
		// this should look like storage/actors/qwazix/nvjfdjelkjdjk.json
		// or storage/actors/qwazix/qwazix.json
	} else {
		// replace slashes with smileys (no, pherephone doesn't work in non unicode filesystems)
		path := makeURLsaveable(strings.Replace(id.String(), baseURL, "", 1))
		jsonFile = storage + slash + "foreign" + slash + path + ".json"
		// this should look like storage/foreign/http:😆😆some.domain😆some😆path.json
	}
	jsonMap, err := readJSON(jsonFile)
	if err != nil {
		log.Info("probably the item doesn't exist in our database")
		return
	}
	value, err = streams.ToType(c, jsonMap)
	if err != nil {
		log.Info("something is wrong with the conversion of JSON to vocab.Type")
		return
	}
	return
}

// Create saves a post to disk
func (d *database) Create(c context.Context, asType vocab.Type) (err error) {
	log.Info("call to create with " + asType.GetActivityStreamsId().GetIRI().String())
	serialized, _ := asType.Serialize()
	json, _ := json.MarshalIndent(serialized, "", "\t")
	// get the id from the activityStreams type
	id := asType.GetActivityStreamsId().Get()

	// if any of our actors own this
	var actor, hash, filename string
	if owns, _ := d.Owns(c, id); owns {
		// split the url and put it in the respective actor
		actor, hash = d.parseIRI(id)
		filename = storage + slash + "actors" + slash + actor + slash + hash + ".json"
	} else {
		// replace slashes with smileys and put it in the `foreign` directory
		filepart := makeURLsaveable(asType.GetActivityStreamsId().Get().String())
		filename = storage + slash + "foreign" + slash + filepart + ".json"
	}

	// actually write the file
	err = ioutil.WriteFile(filename, json, 0644)
	if err != nil {
		log.Printf("Unable to write outbox JSON to file: %+v", err)
		return err
	}
	return
}

// Update does nothing, TODO
func (d *database) Update(c context.Context, asType vocab.Type) (err error) {
	//log.Info("db")
	return
}

// Delete does nothing TODO
func (d *database) Delete(c context.Context, id *url.URL) (err error) {
	//log.Info("db")
	return
}

// GetOutbox actually unserializes the outbox json back to vocab.ActivityStreamsOrderedCollectionPage
func (d *database) GetOutbox(c context.Context, outboxIRI *url.URL) (outbox vocab.ActivityStreamsOrderedCollectionPage, err error) {
	log.Info("getOutbox")
	outbox = streams.NewActivityStreamsOrderedCollectionPage()
	// maybe we should get actor by parsing the IRI here and at some point get rid of `grandparent` TODO
	jsonFile := storage + slash + "actors" + slash + d.grandparent.name + slash + "outbox.json"
	orderedItems := streams.NewActivityStreamsOrderedItemsProperty()
	outbox.SetActivityStreamsOrderedItems(orderedItems)
	outboxData, err := readJSON(jsonFile)
	if err != nil {
		log.Info("Couldn't load outbox, creating...")
		d.SetOutbox(c, outbox)
		// we handled the error so nil
		return outbox, nil
	}
	// if there are already items in the list
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

// SetOutbox is being fed with the new outbox and we have to compare it with the
// old outbox and implement the differences. In our case we just overwrite it
// because we don't have any structured data.
func (d *database) SetOutbox(c context.Context, outbox vocab.ActivityStreamsOrderedCollectionPage) error {
	log.Info("db.SetOutbox")
	serialized, _ := outbox.Serialize()
	json, _ := json.MarshalIndent(serialized, "", "\t")

	log.Info("Creating outbox for " + d.grandparent.name)
	// the actor ought to exist otherwise something is really wrong
	_, err := os.Stat(storage + slash + "actors" + slash + d.grandparent.name)
	if err != nil {
		log.Info("Can't access actor diretory, something is wrong")
		return err
	}
	// just write the serialized outbox to the file as is
	err = ioutil.WriteFile(storage+slash+"actors"+slash+d.grandparent.name+slash+"outbox.json", json, 0644)
	if err != nil {
		log.Printf("Unable to write outbox JSON to file: %+v", err)
		return err
	}

	return err
}

// Followers reads <actor>.json and returns the followers in a Collection
func (d *database) Followers(c context.Context, actorIRI *url.URL) (followers vocab.ActivityStreamsCollection, err error) {
	items := streams.NewActivityStreamsItemsProperty()
	for follower := range d.grandparent.followers {
		iri, _ := url.Parse(follower)
		items.AppendIRI(iri)
	}
	followers.SetActivityStreamsItems(items)
	return
}

// Following reads <actor>.json and returns the actors we are following in a Collection
func (d *database) Following(c context.Context, actorIRI *url.URL) (followers vocab.ActivityStreamsCollection, err error) {
	items := streams.NewActivityStreamsItemsProperty()
	for following := range d.grandparent.following {
		iri, _ := url.Parse(following)
		items.AppendIRI(iri)
	}
	followers.SetActivityStreamsItems(items)
	return
}

// We don't support likes. This returns an empty Collection just to satisfy the interface
func (d *database) Liked(c context.Context, actorIRI *url.URL) (followers vocab.ActivityStreamsCollection, err error) {
	//log.Info("db")
	return
}
