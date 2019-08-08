package main

import (
	"context"
	"net/url"

	"github.com/go-fed/activity/streams"
	"github.com/go-fed/activity/streams/vocab"

	"fmt"
	"encoding/json"
)

type database struct {
}

func (d *database) Lock(c context.Context, id *url.URL) error {
	return nil
}

func (d *database) Unlock(c context.Context, id *url.URL) error {
	return nil
}

func (d *database) NewId(c context.Context, t vocab.Type) (id *url.URL, err error) {
	return
}

func (d *database) InboxContains(c context.Context, inbox, id *url.URL) (contains bool, err error) {
	return
}

func (d *database) GetInbox(c context.Context, inboxIRI *url.URL) (inbox vocab.ActivityStreamsOrderedCollectionPage, err error) {
	fmt.Println("getInboxdb")
	inbox = streams.NewActivityStreamsOrderedCollectionPage()
	return
}

func (d *database) SetInbox(c context.Context, inbox vocab.ActivityStreamsOrderedCollectionPage) error {
	var err error
	return err
}

func (d *database) Owns(c context.Context, id *url.URL) (owns bool, err error) {
	return
}

func (d *database) ActorForOutbox(c context.Context, outboxIRI *url.URL) (actorIRI *url.URL, err error) {
	return
}

func (d *database) ActorForInbox(c context.Context, inboxIRI *url.URL) (actorIRI *url.URL, err error) {
	return
}

func (d *database) OutboxForInbox(c context.Context, inboxIRI *url.URL) (outboxIRI *url.URL, err error) {
	return
}

func (d *database) Exists(c context.Context, id *url.URL) (exists bool, err error) {
	return
}

func (d *database) Get(c context.Context, id *url.URL) (value vocab.Type, err error) {
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

	  if err != nil{
		  fmt.Println("something is wrong with the conversion of JSON to vocab.Type")
		  return
	  }
	
	return
}

func (d *database) Create(c context.Context, asType vocab.Type) (err error) {
	return
}

func (d *database) Update(c context.Context, asType vocab.Type) (err error) {
	return
}

func (d *database) Delete(c context.Context, id *url.URL) (err error) {
	return
}

func (d *database) GetOutbox(c context.Context, outboxIRI *url.URL) (outbox vocab.ActivityStreamsOrderedCollectionPage, err error) {
	outbox = streams.NewActivityStreamsOrderedCollectionPage()
	return
}

func (d *database) SetOutbox(c context.Context, outbox vocab.ActivityStreamsOrderedCollectionPage) error {
	var err error
	return err
}

func (d *database) Followers(c context.Context, actorIRI *url.URL) (followers vocab.ActivityStreamsCollection, err error) {
	return
}

func (d *database) Following(c context.Context, actorIRI *url.URL) (followers vocab.ActivityStreamsCollection, err error) {
	return
}

func (d *database) Liked(c context.Context, actorIRI *url.URL) (followers vocab.ActivityStreamsCollection, err error) {
	return
}
