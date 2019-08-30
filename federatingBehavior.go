package main

import (
	"context"
	"errors"

	"net/http"
	"net/url"

	"strings"

	"github.com/go-fed/activity/pub"
	"github.com/go-fed/activity/streams"
	"github.com/go-fed/activity/streams/vocab"

	"github.com/gologme/log"
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

// This handles inbox requests. Some inbox requests (such as follows) are supposed to be handled by go-fed but I don't know how to trigger
// their callbacks or modify their behavior. For example while there is mentioning of automatically accepting the follows inside the go-fed 
// comments, I can't see how followers are stored in the database.
// Anyway since I had already written much of the following logic and since go-fed chokes on some non-standard input (like writefreely's 
// activities with missing @context) which I want to handle, I'm re handling some stuff here. The errors are still being thrown elsewhere
// in go-fed even if the input is chewed here (so if you see "Can't parse actor, no @context", it might actually still get parsed here)
func (f *federatingBehavior) PostInboxRequestBodyHook(c context.Context, r *http.Request, activity pub.Activity) (out context.Context, err error) {
	// it's a post of some kind, boost it
	if activity.GetTypeName() == "Create" {
		object := activity.GetActivityStreamsObject()
		// check if we are following the author. If we don't
		// it means that we relay whatever comes in and we might be
		// a vehicle for spam
		author := activity.GetActivityStreamsActor().Begin().GetIRI().String()
		// check if we are following this actor and if not bail out
		if f.parent.following[author] == nil {
			return
		}
		// don't announce if it's a reply unless the option in config,ini is
		// set to true
		var serializedObject interface{} // if I use := below err is shadowed
		serializedObject, err = object.Serialize()
		serializedObjectMap := serializedObject.(map[string]interface{})
		if err != nil {
			log.Error("cannot serialize object")
			return
		}
		// serialize the object since aparently `inReplyTo` is non-standard
		// and go-fed has no idea of it
		inReplyTo, ok := serializedObjectMap["inReplyTo"]
		log.Info("Checking if it is a reply")
		log.Info(serializedObjectMap);
		isReply := false;
		// if the field exists and is not null and is not empty
		// then it's a reply
		if ok && inReplyTo != nil && inReplyTo != "" {
			isReply = true;
		}
		// if it's a reply and announce_replies config option
		// is set to false then bail out
		if announceReplies == false && isReply == true {
			return
		}
		// else announce it
		id := object.Begin().GetType().GetActivityStreamsId()
		f.parent.Announce(id.GetIRI().String())
	} else if activity.GetTypeName() == "Follow" {
		// it's a follow, write it down
		actor := activity.GetActivityStreamsActor()
		newFollower := actor.Begin().GetIRI().String()
		// check we aren't following ourselves
		if newFollower == f.parent.iri {
			log.Info("You can't follow yourself")
			return out, errors.New("You can't follow yourself")
		}
		// check if this user is already following us
		if _, ok := f.parent.followers[newFollower]; ok {
			log.Info("You're already following us, yay!")
			// do nothing, they're already following us
		} else {
			f.parent.JotFollowerDown(newFollower)
		}
		// send accept anyway even if they are following us already
		// this is very verbose. I would prefer creating a map by hand
		accept := streams.NewActivityStreamsAccept()
		sender := streams.NewActivityStreamsActorProperty()
		to := streams.NewActivityStreamsToProperty()
		to.AppendIRI(activity.GetActivityStreamsActor().Begin().GetIRI())
		sender.AppendIRI(f.parent.nuIri)
		object := streams.NewActivityStreamsObjectProperty()
		asActivity := streams.NewActivityStreamsActivity()
		asActivity.SetActivityStreamsActor(activity.GetActivityStreamsActor())
		asActivity.SetActivityStreamsId(activity.GetActivityStreamsId())
		asActivity.SetActivityStreamsObject(activity.GetActivityStreamsObject())
		asActivity.SetActivityStreamsTo(activity.GetActivityStreamsTo())
		typename := streams.NewActivityStreamsTypeProperty()
		typename.AppendXMLSchemaString(activity.GetTypeName())
		asActivity.SetActivityStreamsType(typename)
		object.AppendActivityStreamsActivity(asActivity)
		accept.SetActivityStreamsObject(object)
		accept.SetActivityStreamsActor(sender)
		id := streams.NewActivityStreamsIdProperty()
		idIRI, _ := f.db.NewId(c, accept)
		id.SetIRI(idIRI)
		accept.SetActivityStreamsId(id)
		accept.SetActivityStreamsTo(to)
		// log.Info(accept.Serialize())
		go f.parent.pubActor.Send(c, f.parent.GetOutboxIRI(), accept)
	} else if activity.GetTypeName() == "Accept" {
		acceptor := activity.GetActivityStreamsActor()
		// uncomment below to print debugging information
		// ===============================================
		// follow := activity.GetActivityStreamsObject()
		// serializedFollow, _ := follow.Serialize()
		// serializedFollowMap := serializedFollow.(map[string]interface{})
		// log.Info(serializedFollowMap["actor"])
		// actor := follow.GetActivityStreamsActor()
		// acceptorIRI := acceptor.Begin().GetIRI()
		// log.Info(acceptorIRI.String())HandleInbox
		object, _ := activity.GetActivityStreamsObject().Serialize()
		obj := object.(map[string]interface{})
		f.parent.following[acceptor.Begin().GetIRI().String()] = strings.Replace(obj["id"].(string), baseURL+f.parent.name+"/", "", 1)
		f.parent.save()
	} else if activity.GetTypeName() == "Reject" { // handle rejections
		rejector := activity.GetActivityStreamsActor()
		// write the actor to the list of rejected follows so that 
		// we won't try following them again
		f.parent.rejected[rejector.Begin().GetIRI().String()] = ""
		f.parent.save()
	} else if activity.GetTypeName() == "Undo" { // handle unfollowing
		// Serializing and using the map instead of official go-fed types
		object, _ := activity.GetActivityStreamsObject().Serialize()
		obj := object.(map[string]interface{})
		// only if they are undoing a follow
		if obj["type"].(string) == "Follow" {
			requester := obj["actor"].(string)
			delete(f.parent.followers, requester)
			f.parent.save()
		}
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
	log.Info("getInbox")
	var inboxIRI *url.URL
	ocp, err = f.db.GetInbox(c, inboxIRI)
	return
}
