package main

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"time"

	// "log"
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
	// "github.com/go-fed/activity/streams/vocab"

	"github.com/go-fed/activity/pub"
	"github.com/gologme/log"
)

var slash = string(os.PathSeparator)

// Actor represents a local actor we can act on
// behalf of.
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
		log.Info("Something went wrong when parsing the local actor uri into net/url")
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

	publicKeyBlock := pem.Block{
		Type:    "PUBLIC KEY",
		Headers: nil,
		Bytes:   publicKeyDer,
	}
	actor.publicKeyPem = string(pem.EncodeToMemory(&publicKeyBlock))

	err = actor.save()
	if err != nil {
		return actor, err
	}

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

// save the actor to file
func (a *Actor) save() error {

	// check if we already have a directory to save actors
	// and if not, create it
	if _, err := os.Stat(storage + slash + "actors" + slash + a.name); os.IsNotExist(err) {
		os.MkdirAll(storage+slash+"actors"+slash+a.name+slash, 0755)
	}

	// marshal the crypto to json

	// publicKey, _ := json.Marshal(a.publicKey)
	// privateKey, _ := json.Marshal(a.privateKey)

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

	actorJSON, err := json.MarshalIndent(actorToSave, "", "\t")
	if err != nil {
		log.Info("error Marshalling actor json")
		return err
	}
	// log.Info(actorToSave)
	// log.Info(string(actorJSON))
	err = ioutil.WriteFile(storage+slash+"actors"+slash+a.name+slash+a.name+".json", actorJSON, 0644)
	if err != nil {
		log.Printf("WriteFileJson ERROR: %+v", err)
		return err
	}

	// save pubActor to a separate file
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
// from the data in name.json
func LoadActor(name string) (Actor, error) {
	// make sure our users can't read our hard drive
	if strings.ContainsAny(name, "./ ") {
		log.Info("Illegal characters in actor name")
		return Actor{}, errors.New("Illegal characters in actor name")
	}
	jsonFile := storage + slash + "actors" + slash + name + slash + name + ".json"
	fileHandle, err := os.Open(jsonFile)
	if os.IsNotExist(err) {
		log.Info("We don't have this kind of actor stored")
		return Actor{}, err
	}
	byteValue, err := ioutil.ReadAll(fileHandle)
	if err != nil {
		log.Info("Error reading actor file")
		return Actor{}, err
	}
	jsonData := make(map[string]interface{})
	json.Unmarshal(byteValue, &jsonData)

	pubActor, federating, common, db := newPubActor()
	nuIri, err := url.Parse(jsonData["IRI"].(string))
	if err != nil {
		log.Info("Something went wrong when parsing the local actor uri into net/url")
		return Actor{}, err
	}

	// publicKeyNewLines := strings.ReplaceAll(jsonData["PublicKey"].(string), "\\n", "\n")
	// privateKeyNewLines := strings.ReplaceAll(jsonData["PrivateKey"].(string), "\\n", "\n")

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
		log.Info("something is wrong when parsing the remote" +
			"actors iri into a url")
		log.Info(err)
		return err
	}
	to.AppendIRI(iri)
	object.AppendIRI(iri)

	// add "from" actor
	iri, err = url.Parse(a.iri)
	if err != nil {
		log.Info("something is wrong when parsing the local" +
			"actors iri into a url")
		log.Info(err)
		return err
	}
	actorProperty.AppendIRI(iri)
	follow.SetActivityStreamsObject(object)
	follow.SetActivityStreamsTo(to)
	follow.SetActivityStreamsActor(actorProperty)

	// TODO: maybe we need to add and ID property here too
	// go-fed seems to require it, writefreely doesn't

	// log.Info(c)
	// log.Info(iri)
	log.Info(follow.Serialize())
	
	if _, ok := a.following[user]; !ok {
		go func() {
			_, err := a.pubActor.Send(c, iri, follow)
			if err != nil {
				log.Info("Couldn't follow " + user)
				log.Info(err)
				return
			}
			// we are going to save only on accept

			// a.following[user] = struct{}{}
			// a.save()
		}()
	}

	return nil
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
			log.Info("This url is mangled: " + follower + ", ignoring")
		} else {
			cc.AppendIRI(followerIRI)
		}
	}

	publishedProperty := streams.NewActivityStreamsPublishedProperty()
	publishedProperty.Set(time.Now())

	announce.SetActivityStreamsActor(actorProperty)
	announce.SetActivityStreamsObject(objectProperty)
	announce.SetActivityStreamsPublished(publishedProperty)
	announce.SetActivityStreamsCc(cc)
	announce.SetActivityStreamsTo(to)

	go a.pubActor.Send(c, a.nuIri, announce)

	return nil
}

func (a *Actor) whoAmI() string {
	return `{"@context":	"https://www.w3.org/ns/activitystreams",
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

// HandleOutbox handles the outbox of our actor. It actually just
// delegates to go-fed without doing anything in particular.
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
func (a *Actor) HandleInbox(w http.ResponseWriter, r *http.Request) {
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

// func (a *Actor) savePost()

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
