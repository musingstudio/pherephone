package main

import (
	// "github.com/go-fed/activity/streams"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/go-fed/activity/pub"
	"github.com/go-fed/activity/streams/vocab"
	// "strings"
)

type federatingBehavior struct {
	db     *database
	parent *Actor
}

func newFederatingBehavior(db *database) *federatingBehavior {
	return &federatingBehavior{
		db: db,
	}
}

func (f *federatingBehavior) PostInboxRequestBodyHook(c context.Context, r *http.Request, activity pub.Activity) (out context.Context, err error) {
	fmt.Println("postinbox")

	// it's a post of some kind, boost it
	if activity.GetTypeName() == "Create" {
		object := activity.GetActivityStreamsObject()
		// assume it's an article
		// TODO: check what it is and boost it anyway
		article := object.Begin().GetActivityStreamsArticle()
		id := article.GetActivityStreamsId()
		fmt.Println(id)
		f.parent.Announce(id.GetIRI().String())
	} else if activity.GetTypeName() == "Follow" {
		// it's a follow, write it down
		actor := activity.GetActivityStreamsActor()
		newFollower := actor.Begin().GetIRI().String()
		// check we aren't following ourselves
		if newFollower == f.parent.iri {
			fmt.Println("You can't follow yourself")
			return out, errors.New("You can't follow yourself")
		}
		// check if this user is already following us
		if _, ok := f.parent.followers[newFollower]; ok {
			fmt.Println("You're already following us, yay!")
			// do nothing, they're already following us
			return
		}
		f.parent.JotFollowerDown(newFollower)
	}
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
