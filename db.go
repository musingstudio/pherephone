package main

import (
	"context"
	"github.com/go-fed/activity/streams/vocab"
	"github.com/go-fed/activity/streams"
	"net/url"

	"fmt"
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
