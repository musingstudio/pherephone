package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/go-fed/activity/streams"

	"github.com/go-fed/activity/pub"
)

// Actor represents a local actor we can act on
// behalf of.
type Actor struct {
	name, summary, actorType, iri string
	pubActor                      pub.FederatingActor
	nuIri                         *url.URL
}

// MakeActor returns a new local actor we can act
// on behalf of
func MakeActor(name, summary, actorType, iri string) (Actor, error) {
	var clock *clock
	var err error
	var db *database

	clock, err = newClock("Europe/Athens")
	if err != nil {
		return Actor{}, err
	}

	common := newCommonBehavior(db)
	federating := newFederatingBehavior(db)
	pubActor := pub.NewFederatingActor(common, federating, db, clock)
	nuIri, err := url.Parse(iri)
	if err != nil {
		fmt.Println("Something went wrong when parsing the local actor uri into net/url")
		return Actor{}, err
	}
	actor := Actor{
		pubActor:  pubActor,
		name:      name,
		summary:   summary,
		actorType: actorType,
		iri:       iri,
		nuIri:     nuIri,
	}
	federating.parent = &actor
	return actor, nil
}

// Follow a remote user by their iri
// TODO: check if we are already following them
func (a *Actor) Follow(user string) error {
	c := context.Background()

	follow := streams.NewActivityStreamsFollow()
	object := streams.NewActivityStreamsObjectProperty()
	to := streams.NewActivityStreamsToProperty()
	actorProperty := streams.NewActivityStreamsActorProperty()
	iri, err := url.Parse(user)
	// iri, err := url.Parse("https://print3d.social/users/qwazix/outbox")
	if err != nil {
		fmt.Println("something is wrong when parsing the remote" +
			"actors iri into a url")
		fmt.Println(err)
		return err
	}
	to.AppendIRI(iri)
	object.AppendIRI(iri)

	// add "from" actor
	iri, err = url.Parse(a.iri)
	if err != nil {
		fmt.Println("something is wrong when parsing the local" +
			"actors iri into a url")
		fmt.Println(err)
		return err
	}
	actorProperty.AppendIRI(iri)
	follow.SetActivityStreamsObject(object)
	follow.SetActivityStreamsTo(to)
	follow.SetActivityStreamsActor(actorProperty)

	// fmt.Println(c)
	// fmt.Println(iri)
	// fmt.Println(follow)

	go a.pubActor.Send(c, iri, follow)

	return nil
}

// Announce sends an announcement (boost) to the object
// defined by the `object` url
func (a *Actor) Announce(object string) error {
	c := context.Background()

	announcedIRI, err := url.Parse(object)
	if err != nil {
		fmt.Println("Can't parse object url")
		return err
	}
	activityStreamsPublic, err := url.Parse("https://www.w3.org/ns/activitystreams#Public")

	followers, err := url.Parse("http://writefreely.xps/api/collections/qwazix")
	if err != nil {
		fmt.Println("Can't parse follower url")
		return err
	}
	announce := streams.NewActivityStreamsAnnounce()
	objectProperty := streams.NewActivityStreamsObjectProperty()
	objectProperty.AppendIRI(announcedIRI)
	actorProperty := streams.NewActivityStreamsActorProperty()
	actorProperty.AppendIRI(a.nuIri)
	to := streams.NewActivityStreamsToProperty()
	to.AppendIRI(activityStreamsPublic)
	cc := streams.NewActivityStreamsCcProperty()
	cc.AppendIRI(followers)
	announce.SetActivityStreamsActor(actorProperty)
	announce.SetActivityStreamsObject(objectProperty)
	announce.SetActivityStreamsCc(cc)
	announce.SetActivityStreamsTo(to)

	go a.pubActor.Send(c, a.nuIri, announce)

	return nil
}

func (a *Actor) whoAmI() string {
	return `{"@context":	"https://www.w3.org/ns/activitystreams",
	"type": "Person",
	"id": "http://floorb.qwazix.com/actor/",
	"name": "Alyssa P. Hacker",
	"preferredUsername": "pherephone",
	"summary": "pherephone is somebody that repeats others",
	"inbox": "http://floorb.qwazix.com/actor/inbox/",
	"outbox": "http://floorb.qwazix.com/actor/outbox/",
	"followers": "http://floorb.qwazix.com/actor/followers/",
	"following": "http://floorb.qwazix.com/actor/following/",
	"liked": "http://floorb.qwazix.com/` + a.name + `/liked/"}`
}

// HandleOutbox handles the outbox of our actor. It actually just
// delegates to go-fed without doing anything in particular.
func (a *Actor) HandleOutbox(w http.ResponseWriter, r *http.Request) {
	c := context.Background()
	if handled, err := a.pubActor.PostOutbox(c, w, r); err != nil {
		// Write to w
		return
	} else if handled {
		return
	} else if handled, err = a.pubActor.GetOutbox(c, w, r); err != nil {
		// Write to w
		return
	} else if handled {
		fmt.Println("gethandled")
		return
	}
}

// HandleInbox handles the outbox of our actor. It actually just
// delegates to go-fed without doing anything in particular.
func (a *Actor) HandleInbox(w http.ResponseWriter, r *http.Request) {
	c := context.Background()
	if handled, err := a.pubActor.PostInbox(c, w, r); err != nil {
		fmt.Println(err)
		// Write to w
		return
	} else if handled {
		return
	} else if handled, err = a.pubActor.GetInbox(c, w, r); err != nil {
		// Write to w
		return
	} else if handled {
		return
	}
}
