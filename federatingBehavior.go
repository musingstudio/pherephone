package main

import (
	// "github.com/go-fed/activity/streams"
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/go-fed/activity/pub"
	"github.com/go-fed/activity/streams/vocab"

	// "strings"
)

type federatingBehavior struct {
	db *database
}

func newFederatingBehavior(db *database) *federatingBehavior {
	return &federatingBehavior{
		db: db,
	}
}

func (f *federatingBehavior) PostInboxRequestBodyHook(c context.Context, r *http.Request, activity pub.Activity) (out context.Context, err error) {
	object := activity.GetActivityStreamsObject()
	article := object.Begin().GetActivityStreamsArticle()
	id := article.GetActivityStreamsId()
	fmt.Println(id)

	// parts := strings.Split(r.RequestURI, "/")
	// the last part (https://example.com/actor/qwazix)
	// actorName := parts[len(parts)-1]

	// TODO: select name and stuff from database according to URI

	actor, err := MakeActor("Pherephone", "pherephone repeats", "service", domainName+r.RequestURI)
	if err != nil{
		fmt.Println("Couldn't create local actor")
		return
	}

	actor.Announce(id.GetIRI().String())
	return
}

func (f *federatingBehavior) AuthenticatePostInbox(c context.Context, w http.ResponseWriter, r *http.Request) (authenticated bool, err error) {
	// TODO
	// 1. Validate HTTP Signatures
	return true, nil
}

func (f *federatingBehavior) Blocked(c context.Context, actorIRIs []*url.URL) (blocked bool, err error) {
	return
}

func (f *federatingBehavior) Callbacks(c context.Context) (wrapped pub.FederatingWrappedCallbacks, other []interface{}, err error) {
	return
}

func (f *federatingBehavior) DefaultCallback(c context.Context, activity pub.Activity) error {
	return nil
}

func (f *federatingBehavior) MaxInboxForwardingRecursionDepth(c context.Context) int {
	return 100
}

func (f *federatingBehavior) MaxDeliveryRecursionDepth(c context.Context) int {
	return 100
}

func (f *federatingBehavior) FilterForwarding(c context.Context, potentialRecipients []*url.URL, a pub.Activity) (filteredRecipients []*url.URL, err error) {
	// TODO
	return
}

func (f *federatingBehavior) GetInbox(c context.Context, r *http.Request) (ocp vocab.ActivityStreamsOrderedCollectionPage, err error) {
	fmt.Println("getInbox")
	var inboxIRI *url.URL
	ocp, err = f.db.GetInbox(c, inboxIRI)
	return
}
