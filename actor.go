package main

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"time"

	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"

	"github.com/go-fed/activity/streams"
	"github.com/go-fed/activity/streams/vocab"

	"github.com/go-fed/activity/pub"
	"github.com/gologme/log"
)

var slash = string(os.PathSeparator)

// Actor represents a local actor we can act on
// behalf of. This contains a PubActor which is
// an instance of the FederatingActor go-fed interface
type Actor struct {
	name, summary, actorType, iri string
	pubActor                      pub.FederatingActor
	nuIri                         *url.URL
	followers, following          map[string]interface{}
	posts                         map[int]map[string]string
	publicKey                     crypto.PublicKey
	privateKey                    crypto.PrivateKey
	publicKeyPem                  string
	privateKeyPem                 string
}

// ActorToSave is a stripped down actor representation
// with exported properties in order for json to be
// able to marshal it.
// see https://stackoverflow.com/questions/26327391/json-marshalstruct-returns
type ActorToSave struct {
	Name, Summary, ActorType, IRI, PublicKey, PrivateKey string
	Followers, Following                                 map[string]interface{}
}

// newPubActor constructs a go-fed federating actor with all the required components
func newPubActor() (pub.FederatingActor, *commonBehavior, *federatingBehavior, *database) {
	var clock *clock
	var err error
	db := new(database)

	clock, _ = newClock("Europe/Athens")
	if err != nil {
		log.Info("error creating clock")
	}

	common := newCommonBehavior(db)
	federating := newFederatingBehavior(db)
	pubActor := pub.NewFederatingActor(common, federating, db, clock)

	//kludgey, but we need common and federating to set their parents
	//can't think of a better architecture for now
	return pubActor, common, federating, db
}

// MakeActor returns a new local actor we can act
// on behalf of
func MakeActor(name, summary, actorType, iri string) (Actor, error) {
	pubActor, common, federating, db := newPubActor()
	// We store the followers in the key so that we
	// get free deduplication and easy search
	// The value is populated with the hash part (the thing
	// after the last slash) of the id of the Follow activity
	// that created the relationship
	followers := make(map[string]interface{})
	following := make(map[string]interface{})
	// nuIri is the actor IRI in net/url format instead of string
	nuIri, err := url.Parse(iri)
	if err != nil {
		log.Info("Something went wrong when parsing the local actor uri into net/url")
		return Actor{}, err
	}
	// we compose the actor here so that we can go afterwards
	// and create some pointers to it inside it's children
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

	// create actor's keypair
	rng := rand.Reader
	privateKey, err := rsa.GenerateKey(rng, 2048)
	publicKey := privateKey.PublicKey

	actor.publicKey = publicKey
	actor.privateKey = privateKey

	// marshal the crypto to pem
	privateKeyDer := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyBlock := pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   privateKeyDer,
	}
	actor.privateKeyPem = string(pem.EncodeToMemory(&privateKeyBlock))

	publicKeyDer, err := x509.MarshalPKIXPublicKey(&publicKey)
	if err != nil {
		log.Info("Can't marshal public key")
		return Actor{}, err
	}

	// create the --- BEGIN PUBLIC KEY --- block
	publicKeyBlock := pem.Block{
		Type:    "PUBLIC KEY",
		Headers: nil,
		Bytes:   publicKeyDer,
	}
	actor.publicKeyPem = string(pem.EncodeToMemory(&publicKeyBlock))

	// save the actor to file. This file is sensitive as
	// it contains the private keys
	err = actor.save()
	if err != nil {
		return actor, err
	}

	// pass pointers to this actor to the go-fed interfaces
	// so they can call stuff without parsing the IRI's every
	// time.

	// This doesn't look like following the philosophy of go-fed
	// but I'm not really sure I understand its philosophy given
	// the lack of documentation
	federating.parent = &actor
	common.parent = &actor
	db.grandparent = &actor
	return actor, nil
}

// GetOutboxIRI returns the outbox iri in net/url
func (a *Actor) GetOutboxIRI() *url.URL {
	iri := a.iri + "/outbox"
	nuiri, _ := url.Parse(iri)
	return nuiri
}

// save the actor to file. This file is sensitive
// as it contains the private key
func (a *Actor) save() error {

	// check if we already have a directory to save actors
	// and if not, create it
	// The directory looks like ./storage/actors/thisActor/
	if _, err := os.Stat(storage + slash + "actors" + slash + a.name); os.IsNotExist(err) {
		os.MkdirAll(storage+slash+"actors"+slash+a.name+slash, 0755)
	}

	// fill the struct to be saved with stuff from the actor
	actorToSave := ActorToSave{
		Name:       a.name,
		Summary:    a.summary,
		ActorType:  a.actorType,
		IRI:        a.iri,
		Followers:  a.followers,
		Following:  a.following,
		PublicKey:  a.publicKeyPem,
		PrivateKey: a.privateKeyPem,
	}

	// marshal to JSON
	actorJSON, err := json.MarshalIndent(actorToSave, "", "\t")
	if err != nil {
		log.Info("error Marshalling actor json")
		return err
	}

	// log.Info(actorToSave)
	// log.Info(string(actorJSON))

	// Write the actual file
	err = ioutil.WriteFile(storage+slash+"actors"+slash+a.name+slash+a.name+".json", actorJSON, 0644)
	if err != nil {
		log.Printf("WriteFileJson ERROR: %+v", err)
		return err
	}

	// save the ActivityPub representation to a separate file
	// this file is not sensitive and it contains the public actor JSON.
	// This might be redundant.
	// TODO: investigat the possibility of deleting this.
	actorJSON = []byte(a.whoAmI())
	err = ioutil.WriteFile(storage+slash+"actors"+slash+a.name+slash+"actor.json", actorJSON, 0644)
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
		log.Info("Actor doesn't exist, creating...")
		actor, err = MakeActor(name, summary, actorType, iri)
		if err != nil {
			log.Info("Can't create actor!")
			return Actor{}, err
		}
	}
	return actor, nil
}

// LoadActor searches the filesystem and creates an Actor
// from the data in <name>.json
func LoadActor(name string) (Actor, error) {
	// make sure our users can't read our hard drive
	if strings.ContainsAny(name, "./ ") {
		log.Info("Illegal characters in actor name")
		return Actor{}, errors.New("Illegal characters in actor name")
	}

	// search storage/actors/<name>/<name>.json
	jsonFile := storage + slash + "actors" + slash + name + slash + name + ".json"
	fileHandle, err := os.Open(jsonFile)
	if os.IsNotExist(err) {
		// if it doesn't exist, give up
		log.Info("We don't have this kind of actor stored: "+ name)
		return Actor{}, err
	}
	// read the file
	byteValue, err := ioutil.ReadAll(fileHandle)
	if err != nil {
		log.Info("Error reading actor file")
		return Actor{}, err
	}
	// unmarshal it to json
	jsonData := make(map[string]interface{})
	json.Unmarshal(byteValue, &jsonData)

	// Start setting up stuff so that we can create an Actor

	// create a new pubActor to pass to the newly created Actor
	pubActor, federating, common, db := newPubActor()
	// parse it's IRI to net/url
	nuIri, err := url.Parse(jsonData["IRI"].(string))
	if err != nil {
		log.Info("Something went wrong when parsing the local actor uri into net/url")
		return Actor{}, err
	}

	// Unmarshal the keys to crypto.xxxxkey
	publicKeyDecoded, rest := pem.Decode([]byte(jsonData["PublicKey"].(string)))
	if publicKeyDecoded == nil {
		log.Info(rest)
		panic("failed to parse PEM block containing the public key")
	}
	publicKey, err := x509.ParsePKIXPublicKey(publicKeyDecoded.Bytes)
	if err != nil {
		log.Info("Can't parse public keys")
		log.Info(err)
		return Actor{}, err
	}
	privateKeyDecoded, rest := pem.Decode([]byte(jsonData["PrivateKey"].(string)))
	if privateKeyDecoded == nil {
		log.Info(rest)
		panic("failed to parse PEM block containing the private key")
	}
	privateKey, err := x509.ParsePKCS1PrivateKey(privateKeyDecoded.Bytes)
	if err != nil {
		log.Info("Can't parse private keys")
		log.Info(err)
		return Actor{}, err
	}

	// create the Actor and populate all the properties
	actor := Actor{
		pubActor:      pubActor,
		name:          name,
		summary:       jsonData["Summary"].(string),
		actorType:     jsonData["ActorType"].(string),
		iri:           jsonData["IRI"].(string),
		nuIri:         nuIri,
		followers:     jsonData["Followers"].(map[string]interface{}),
		following:     jsonData["Following"].(map[string]interface{}),
		publicKey:     publicKey,
		privateKey:    privateKey,
		publicKeyPem:  jsonData["PublicKey"].(string),
		privateKeyPem: jsonData["PrivateKey"].(string),
	}

	// give the children pointers to this Actor
	federating.parent = &actor
	common.parent = &actor
	db.grandparent = &actor

	return actor, nil
}

// This is to be reused because unfollowing just wraps
// The follow activity with an Undo activity
// This returns a new "Follow" activity
func (a *Actor) getFollowActivity(user string) (follow vocab.ActivityStreamsFollow, err error) {
	follow = streams.NewActivityStreamsFollow()
	object := streams.NewActivityStreamsObjectProperty()
	to := streams.NewActivityStreamsToProperty()
	actorProperty := streams.NewActivityStreamsActorProperty()
	iri, err := url.Parse(user)
	if err != nil {
		log.Info("something is wrong when parsing the remote" +
			"actors iri into a url")
		log.Info(err)
		return
	}
	to.AppendIRI(iri)
	object.AppendIRI(iri)

	// add "from" actor
	iri, err = url.Parse(a.iri)
	if err != nil {
		log.Info("something is wrong when parsing the local" +
			"actors iri into a url")
		log.Info(err)
		return
	}
	actorProperty.AppendIRI(iri)
	follow.SetActivityStreamsObject(object)
	follow.SetActivityStreamsTo(to)
	follow.SetActivityStreamsActor(actorProperty)

	// log.Info(c)
	// log.Info(iri)
	// log.Info(follow.Serialize())
	return
}

// Follow a remote user by their iri
func (a *Actor) Follow(user string) error {
	c := context.Background()

	follow, err := a.getFollowActivity(user)

	if err != nil {
		log.Error("Cannot create follow activity")
		return err
	}

	if _, ok := a.following[user]; !ok {
		go func() {
			_, err := a.pubActor.Send(c, a.GetOutboxIRI(), follow)
			if err != nil {
				log.Info("Couldn't follow " + user)
				log.Info(err)
				return
			}
			// we are going to save only on accept so look at
			// federatingBehavior.go#PostInboxRequestBodyHook()
		}()
	}

	return nil
}

// Unfollow the user declared by the iri in `user`
// this calls getFollowActivity to get a follow
// activity, wraps it in an Undo activity, sets it's
// id to the id of the original Follow activity that
// was accepted when initially following that user
// (this is read from the `actor.following` map
func (a *Actor) Unfollow(user string) {
	c := context.Background()
	log.Info("Unfollowing " + user)

	// create an undo activiy
	undo := streams.NewActivityStreamsUndo()
	actor := streams.NewActivityStreamsActorProperty()
	object := streams.NewActivityStreamsObjectProperty()
	actor.AppendIRI(a.nuIri)

	// find the id of the original follow
	hash := a.following[user].(string)
	followid := streams.NewActivityStreamsIdProperty()
	followidiri, _ := url.Parse(baseURL + a.name + "/" + hash)
	followid.Set(followidiri)

	// create a follow activity
	followActivity, err := a.getFollowActivity(user)
	if err != nil {
		log.Error("Cannot create follow activity")
		return
	}
	object.AppendActivityStreamsFollow(followActivity)

	// set the id to the one we found before
	followActivity.SetActivityStreamsId(followid)

	// add the properties to the undo activity
	undo.SetActivityStreamsObject(object)
	undo.SetActivityStreamsActor(actor)

	// only if we're already following them
	if _, ok := a.following[user]; ok {
		log.Info(undo.Serialize())
		go func() {
			_, err := a.pubActor.Send(c, a.GetOutboxIRI(), undo)
			if err != nil {
				log.Info("Couldn't unfollow " + user)
				log.Info(err)
				return
			}
			// if there was no error then delete the follow
			// from the list
			delete(a.following, user)
			a.save()
		}()
	}
}

// Announce sends an announcement (boost) to the object
// defined by the `object` url
func (a *Actor) Announce(object string) error {
	log.Info(1, "About to announce post with iri "+object)
	c := context.Background()

	announcedIRI, err := url.Parse(object)
	if err != nil {
		log.Info("Can't parse object url")
		return err
	}

	// our announcements are public. Public stuff have a "To" to the url below
	activityStreamsPublic, err := url.Parse("https://www.w3.org/ns/activitystreams#Public")

	announce := streams.NewActivityStreamsAnnounce()
	objectProperty := streams.NewActivityStreamsObjectProperty()
	objectProperty.AppendIRI(announcedIRI)
	actorProperty := streams.NewActivityStreamsActorProperty()
	actorProperty.AppendIRI(a.nuIri)
	to := streams.NewActivityStreamsToProperty()
	to.AppendIRI(activityStreamsPublic)

	// cc this to all our followers one by one
	// I've seen activities to just include the url of the
	// collection but for now this works.
	cc := streams.NewActivityStreamsCcProperty()
	for follower := range a.followers {
		followerIRI, err := url.Parse(follower)
		if err != nil {
			log.Info("This url is mangled: " + follower + ", ignoring")
		} else {
			cc.AppendIRI(followerIRI)
		}
	}

	// add a timestamp
	publishedProperty := streams.NewActivityStreamsPublishedProperty()
	publishedProperty.Set(time.Now())

	announce.SetActivityStreamsActor(actorProperty)
	announce.SetActivityStreamsObject(objectProperty)
	announce.SetActivityStreamsPublished(publishedProperty)
	announce.SetActivityStreamsCc(cc)
	announce.SetActivityStreamsTo(to)

	// send it
	go a.pubActor.Send(c, a.nuIri, announce)

	return nil
}

// whoAmI returns the actor information in ActivityStreams format
// TODO: make this use the streams library
func (a *Actor) whoAmI() string {
	return `{"@context":	["https://www.w3.org/ns/activitystreams"],
	"type": "` + a.actorType + `",
	"id": "` + baseURL + a.name + `",
	"name": "` + a.name + `",
	"preferredUsername": "` + a.name + `",
	"summary": "` + a.summary + `",
	"inbox": "` + baseURL + a.name + `/inbox",
	"outbox": "` + baseURL + a.name + `/outbox",
	"followers": "` + baseURL + a.name + `/followers",
	"following": "` + baseURL + a.name + `/following",
	"liked": "` + baseURL + a.name + `/liked",
	"publicKey": {
		"id": "` + baseURL + a.name + `#main-key",
		"owner": "` + baseURL + a.name + `",
		"publicKeyPem": "` + strings.ReplaceAll(a.publicKeyPem, "\n", "\\n") + `"
	  }
	}`
}

// Load a post with a specific hash (the part after the lash slash of the id IRI)
func (a *Actor) getPost(hash string) (post string, err error) {
	// make sure our users can't read our hard drive
	if strings.ContainsAny(hash, "./ ") {
		log.Info("Illegal characters in post name")
		return "", errors.New("Illegal characters in post name")
	}
	if hash == a.name {
		log.Info("Post id cannot be = to actor name")
		return "", errors.New("Post id cannot be = to actor name")
	}
	filename := storage + slash + "actors" + slash + a.name + slash + hash + ".json"
	post, err = readStringFromFile(filename)
	if err != nil {
		log.Info("this post doesn't exist")
	}
	return
}

// HandleOutbox handles the outbox of our actor. It used to just
// delegate to go-fed without doing anything in particular. (commented code)
// but trying to solve some issues I decided to write the functionality myself
// because go-fed returned an orderedcollection with a bunch of IRI's instead
// of the paged interface mastodon and pixelfed have.
// This didn't actually solve the issue of getting the old boosts when viewing
// the account from mastodon, but it's possible that mastodon only shows the
// posts that at some point federated with them so I gave up.
// However I kept the new layout.
func (a *Actor) HandleOutbox(w http.ResponseWriter, r *http.Request) {
	// c := context.Background()
	// if handled, err := a.pubActor.PostOutbox(c, w, r); err != nil {
	// 	// Write to w
	// 	return
	// } else if handled {
	// 	return
	// } else if handled, err = a.pubActor.GetOutbox(c, w, r); err != nil {
	// 	// Write to w
	// 	return
	// } else if handled {
	// 	log.Info("gethandled")
	// 	return
	// }
	w.Header().Set("content-type", "application/activity+json; charset=utf-8")
	actor, err := LoadActor(a.name) // load the actor from disk
	if err != nil {                 // either actor requested has illegal characters or
		log.Info("Can't load local actor") // we don't have such actor
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("404 - page not found"))
		return
	}
	var response []byte
	page := r.URL.Query().Get("page")
	if page == "" {

		outboxJSON, err := ioutil.ReadFile(storage + slash + "actors" + slash + a.name + slash + "outbox.json")
		if err != nil {
			log.Error("can't read outbox")
			return
		}
		outboxMap := make(map[string]interface{})
		err = json.Unmarshal(outboxJSON, &outboxMap)

		if err != nil {
			log.Error("can't unmarshal outbox")
			return
		}

		response = []byte(`{
			"@context" : "https://www.w3.org/ns/activitystreams",
			"first" : "` + baseURL + actor.name + `/outbox?page=true",
			"id" : "` + baseURL + actor.name + `/outbox",
			"last" : "` + baseURL + actor.name + `/outbox?min_id=0&page=true",
			"totalItems" : ` + strconv.Itoa(len(outboxMap["orderedItems"].([]interface{}))) + `, 
			"type" : "OrderedCollection"
			}`)
	} else if page == "1" {
		collectionPage := make(map[string]interface{})
		collectionPage["@context"] = "https://www.w3.org/ns/activitystreams"
		collectionPage["id"] = baseURL + a.name + "/outbox?page=" + page

		outboxJSON, err := ioutil.ReadFile(storage + slash + "actors" + slash + a.name + slash + "outbox.json")
		if err != nil {
			log.Error("can't read outbox")
			return
		}
		outboxMap := make(map[string]interface{})
		err = json.Unmarshal(outboxJSON, &outboxMap)

		if err != nil {
			log.Error("can't unmarshal outbox")
			return
		}

		items := make([]interface{}, 0, len(outboxMap["orderedItems"].([]interface{})))
		for _, id := range outboxMap["orderedItems"].([]interface{}) {
			parts := strings.Split(id.(string), "/")
			hash := parts[len(parts)-1]
			activityJSON, err := ioutil.ReadFile(storage + slash + "actors" + slash + a.name + slash + hash + ".json")
			if err != nil {
				log.Error("can't read activity")
				return
			}
			var temp map[string]string
			json.Unmarshal(activityJSON, &temp)
			items = append(items, temp)
		}
		collectionPage["orderedItems"] = items
		collectionPage["partOf"] = baseURL + a.name + "/outbox"
		collectionPage["type"] = "OrderedCollectionPage"
		response, _ = json.Marshal(collectionPage)
	}
	w.Write([]byte(response))
}

// HandleInbox handles the inbox of our actor. It actually just
// delegates to go-fed without doing anything in particular.
// As it is now it returns an empty collection. I do not know
// if we need to implement an inbox
func (a *Actor) HandleInbox(w http.ResponseWriter, r *http.Request) {
	// body,_ := ioutil.ReadAll(r.Body)
	// log.Info(string(body))
	// log.Info("&&&&&&&&&&&&&")
	c := context.Background()
	if handled, err := a.pubActor.PostInbox(c, w, r); err != nil {
		log.Info(err)
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

// GetFollowers returns a list of people that follow us
func (a *Actor) GetFollowers(page int) (response []byte, err error) {
	if page == 0 {
		// collection = streams.NewActivityStreamsOrderedCollection()
		// totalItems := streams.NewActivityStreamsTotalItemsProperty()
		// totalItems.Set(len(a.followers))
		// collection.SetActivityStreamsTotalItems(totalItems)
		// first := streams.NewActivityStreamsFirstProperty()
		// firstIRI := url.Parse()
		// first.SetIRI()

		response = []byte(`{
			"@context" : "https://www.w3.org/ns/activitystreams",
			"first" : "` + baseURL + slash + a.name + `/followers?page=1",
			"id" : "` + baseURL + slash + a.name + `/followers",
			"totalItems" : ` + strconv.Itoa(len(a.followers)) + `,
			"type" : "OrderedCollection"
		 }`)
	} else if page == 1 { // implement pagination
		collectionPage := make(map[string]interface{})
		collectionPage["@context"] = "https://www.w3.org/ns/activitystreams"
		collectionPage["id"] = baseURL + slash + a.name + "followers?page=" + strconv.Itoa(page)
		items := make([]string, 0, len(a.followers))
		for k := range a.followers {
			items = append(items, k)
		}
		collectionPage["orderedItems"] = items
		collectionPage["partOf"] = baseURL + slash + a.name + "/followers"
		collectionPage["totalItems"] = len(a.followers)
		collectionPage["type"] = "OrderedCollectionPage"
		response, _ = json.Marshal(collectionPage)
	}
	return
}

// GetFollowing returns a list of people we follow
func (a *Actor) GetFollowing(page int) (response []byte, err error) {
	if page == 0 {
		response = []byte(`{
			"@context" : "https://www.w3.org/ns/activitystreams",
			"first" : "` + baseURL + slash + a.name + `/following?page=1",
			"id" : "` + baseURL + slash + a.name + `/following",
			"totalItems" : ` + strconv.Itoa(len(a.following)) + `,
			"type" : "OrderedCollection"
		 }`)
	} else if page == 1 { // implement pagination
		collectionPage := make(map[string]interface{})
		collectionPage["@context"] = "https://www.w3.org/ns/activitystreams"
		collectionPage["id"] = baseURL + slash + a.name + "following?page=" + strconv.Itoa(page)
		items := make([]string, 0, len(a.following))
		for k := range a.following {
			items = append(items, k)
		}
		collectionPage["orderedItems"] = items
		collectionPage["partOf"] = baseURL + slash + a.name + "/following"
		collectionPage["totalItems"] = len(a.following)
		collectionPage["type"] = "OrderedCollectionPage"
		response, _ = json.Marshal(collectionPage)
	}
	return
}
