package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/go-fed/activity/streams"

	"github.com/go-fed/activity/pub"
)

var slash = string(os.PathSeparator)

// Actor represents a local actor we can act on
// behalf of.
type Actor struct {
	name, summary, actorType, iri	string
	pubActor						pub.FederatingActor
	nuIri							*url.URL
	followers, following			map[string]interface{}
	posts							map[int]map[string]string
}

// ActorToSave is a stripped down actor representation
// with exported properties in order for json to be
// able to marshal it.
// see https://stackoverflow.com/questions/26327391/json-marshalstruct-returns
type ActorToSave struct {
	Name, Summary, ActorType, IRI string
	Followers, Following          map[string]interface{}
}

func newPubActor() (pub.FederatingActor, *commonBehavior, *federatingBehavior, *database) {
	var clock *clock
	var err error
	db := new(database)

	clock, _ = newClock("Europe/Athens")
	if err != nil {
		log.Println("error creating clock")
	}

	common := newCommonBehavior(db)
	federating := newFederatingBehavior(db)
	pubActor := pub.NewFederatingActor(common, federating, db, clock)

	//kludgey, but we need common and federating to set their parents
	//can't think of a better architecture for now
	return pubActor, common, federating, db
}

// // set up and return a pubActor object for our actor
// func (a *Actor) getPubActor() pub.FederatingActor{
// 	// if we already have one return it
// 	if a.pubActor != nil {
// 		return a.pubActor
// 	} // else make a new one
// 	// := cannot mix assingment with declaration so
// 	// I either had to make an extra variable and then
// 	// assign a.pubActor to pubActor or declare the behaviors
// 	// beforehand. I chose the latter
// 	var common *commonBehavior
// 	var federating *federatingBehavior
// 	a.pubActor, common, federating = newPubActor()
// 	// assign our actor pointer to be the parent of
// 	// these two behaviors so that afterwards in e.g.
// 	// GetInbox we can know which actor we are talking
// 	// about
// 	federating.parent = a
// 	common.parent = a
// 	return a.pubActor
// }

// MakeActor returns a new local actor we can act
// on behalf of
func MakeActor(name, summary, actorType, iri string) (Actor, error) {
	pubActor, common, federating, db := newPubActor()
	followers := make(map[string]interface{})
	following := make(map[string]interface{})
	nuIri, err := url.Parse(iri)
	if err != nil {
		log.Println("Something went wrong when parsing the local actor uri into net/url")
		return Actor{}, err
	}
	actor := Actor{
		pubActor:  pubActor,
		name:      name,
		summary:   summary,
		actorType: actorType,
		iri:       iri,
		nuIri:     nuIri,
		followers: followers,
		following: following,
	}

	err = actor.save()
	if err != nil {
		return actor, err
	}

	federating.parent = &actor
	common.parent = &actor
	db.grandparent = &actor
	return actor, nil
}

// save the actor to file
func (a *Actor) save() error {

	// check if we already have a directory to save actors
	// and if not, create it
	if _, err := os.Stat("actors" + slash + a.name); os.IsNotExist(err) {
		os.MkdirAll("actors"+slash+a.name+slash, 0755)
	}

	actorToSave := ActorToSave{
		Name:      a.name,
		Summary:   a.summary,
		ActorType: a.actorType,
		IRI:       a.iri,
		Followers: a.followers,
		Following: a.following,
	}

	actorJSON, err := json.MarshalIndent(actorToSave, "", "\t")
	if err != nil {
		log.Println("error Marshalling actor json")
		return err
	}
	log.Println(actorToSave)
	log.Println(string(actorJSON))
	err = ioutil.WriteFile("actors"+slash+a.name+slash+a.name+".json", actorJSON, 0644)
	if err != nil {
		log.Printf("WriteFileJson ERROR: %+v", err)
		return err
	}
	return nil
}

// GetActor attempts to LoadActor and if it doesn't exist
// creates one
func GetActor(name, summary, actorType, iri string) (Actor, error) {
	actor, err := LoadActor(name)

	if err != nil {
		log.Println("Actor doesn't exist, creating...")
		actor, err = MakeActor(name, summary, actorType, iri)
		if err != nil {
			log.Println("Can't create actor!")
			return Actor{}, err
		}
	}
	return actor, nil
}

// LoadActor searches the filesystem and creates an Actor
// from the data in name.json
func LoadActor(name string) (Actor, error) {
	jsonFile := "actors" + slash + name + slash + name + ".json"
	fileHandle, err := os.Open(jsonFile)
	if os.IsNotExist(err) {
		log.Println("We don't have this kind of actor stored")
		return Actor{}, err
	}
	byteValue, err := ioutil.ReadAll(fileHandle)
	if err != nil {
		log.Println("Error reading actor file")
		return Actor{}, err
	}
	jsonData := make(map[string]interface{})
	json.Unmarshal(byteValue, &jsonData)

	pubActor, federating, common, db := newPubActor()
	nuIri, err := url.Parse(jsonData["IRI"].(string))
	if err != nil {
		log.Println("Something went wrong when parsing the local actor uri into net/url")
		return Actor{}, err
	}

	actor := Actor{
		pubActor:  pubActor,
		name:      name,
		summary:   jsonData["Summary"].(string),
		actorType: jsonData["ActorType"].(string),
		iri:       jsonData["IRI"].(string),
		nuIri:     nuIri,
		followers: jsonData["Followers"].(map[string]interface{}),
		following: jsonData["Following"].(map[string]interface{}),
	}

	federating.parent = &actor
	common.parent = &actor
	db.grandparent = &actor

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
		log.Println("something is wrong when parsing the remote" +
			"actors iri into a url")
		log.Println(err)
		return err
	}
	to.AppendIRI(iri)
	object.AppendIRI(iri)

	// add "from" actor
	iri, err = url.Parse(a.iri)
	if err != nil {
		log.Println("something is wrong when parsing the local" +
			"actors iri into a url")
		log.Println(err)
		return err
	}
	actorProperty.AppendIRI(iri)
	follow.SetActivityStreamsObject(object)
	follow.SetActivityStreamsTo(to)
	follow.SetActivityStreamsActor(actorProperty)

	// TODO: maybe we need to add and ID property here too
	// go-fed seems to require it, writefreely doesn't

	// log.Println(c)
	// log.Println(iri)
	// log.Println(follow)

	if _, ok := a.following[user]; !ok {
		a.following[user] = struct{}{}
		go a.pubActor.Send(c, iri, follow)
		a.save()
	}

	return nil
}

// Announce sends an announcement (boost) to the object
// defined by the `object` url
func (a *Actor) Announce(object string) error {
	log.Println(1, "About to announce post with iri "+object)
	c := context.Background()

	announcedIRI, err := url.Parse(object)
	if err != nil {
		log.Println("Can't parse object url")
		return err
	}
	activityStreamsPublic, err := url.Parse("https://www.w3.org/ns/activitystreams#Public")

	announce := streams.NewActivityStreamsAnnounce()
	objectProperty := streams.NewActivityStreamsObjectProperty()
	objectProperty.AppendIRI(announcedIRI)
	actorProperty := streams.NewActivityStreamsActorProperty()
	actorProperty.AppendIRI(a.nuIri)
	to := streams.NewActivityStreamsToProperty()
	to.AppendIRI(activityStreamsPublic)
	cc := streams.NewActivityStreamsCcProperty()
	for follower := range a.followers {
		followerIRI, err := url.Parse(follower)
		if err != nil {
			log.Println("This url is mangled: " + follower + ", ignoring")
		} else {
			cc.AppendIRI(followerIRI)
		}
	}

	announce.SetActivityStreamsActor(actorProperty)
	announce.SetActivityStreamsObject(objectProperty)
	announce.SetActivityStreamsCc(cc)
	announce.SetActivityStreamsTo(to)

	go a.pubActor.Send(c, a.nuIri, announce)

	return nil
}

func (a *Actor) whoAmI() string {
	return `{"@context":	"https://www.w3.org/ns/activitystreams",
	"type": "` + a.actorType + `",
	"id": "http://floorb.qwazix.com/` + a.name + `/",
	"name": "Alyssa P. Hacker",
	"preferredUsername": "` + a.name + `",
	"summary": "` + a.summary + `",
	"inbox": "http://floorb.qwazix.com/` + a.name + `/inbox/",
	"outbox": "http://floorb.qwazix.com/` + a.name + `/outbox/",
	"followers": "http://floorb.qwazix.com/` + a.name + `/followers/",
	"following": "http://floorb.qwazix.com/` + a.name + `/following/",
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
		log.Println("gethandled")
		return
	}
}

// HandleInbox handles the outbox of our actor. It actually just
// delegates to go-fed without doing anything in particular.
func (a *Actor) HandleInbox(w http.ResponseWriter, r *http.Request) {
	c := context.Background()
	if handled, err := a.pubActor.PostInbox(c, w, r); err != nil {
		log.Println(err)
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

// JotFollowerDown saves the fact that we have been followed to a file
func (a *Actor) JotFollowerDown(iri string) error {
	a.followers[iri] = struct{}{}
	return a.save()
}

// func (a *Actor) savePost()