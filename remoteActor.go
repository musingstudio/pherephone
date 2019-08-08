package main


import (
	"fmt"

	"github.com/go-fed/activity/pub"
	"github.com/go-fed/httpsig"

	"net/url"
	"net/http"

	"encoding/json"

	"context"
	"crypto/rsa"
	"crypto/rand"
)

// RemoteActor is a type that holds an actor 
// that we want to interact with
type RemoteActor struct{
	iri string
	info map[string]interface{}
	outboxIri string
}

// NewRemoteActor returns a remoteActor which holds
// all the info required for an actor we want to 
// interact with (not essentially sitting in our instance)
func NewRemoteActor(iri string) RemoteActor {
	
	info := get(iri)

	outboxIri := info["outbox"].(string)

		return RemoteActor{
			iri: iri,
			outboxIri: outboxIri,
		}
}

func (ra RemoteActor) getLatestPosts(number int) map[string]interface{} {
	return get(ra.outboxIri)
}

func get(iri string) map[string]interface{} {
	clock, err := newClock("Europe/Athens")
	if err != nil {
		fmt.Println("something is wrong with the clock")
		fmt.Println(err)
	}

	client := &http.Client{}

	getSigner, _, err := httpsig.NewSigner( []httpsig.Algorithm{httpsig.RSA_SHA256}, []string{"(request-target)", "date", "host", "digest"}, httpsig.Signature )
	postSigner, _, err := httpsig.NewSigner( []httpsig.Algorithm{httpsig.RSA_SHA256}, []string{"(request-target)", "date", "host", "digest"}, httpsig.Signature )
	if err != nil{
		fmt.Println("something is wrong with the httpsigner function call")
		fmt.Println(err)
	}
	pubKeyId := ""
	rng := rand.Reader
	privKey, err := rsa.GenerateKey(rng, 2048)
	if err != nil{
		fmt.Println("something is wrong with the httpsigner function call")
		fmt.Println(err)
	}

	transport := pub.NewHttpSigTransport(client, "pherephone", clock, getSigner, postSigner, pubKeyId, privKey)

	c := context.Background()
	actorURL, err := url.Parse(iri)
	res, err := transport.Dereference(c, actorURL)
	if err!=nil{
		fmt.Println("something is wrong with the request")
		fmt.Println(err)
	}

	// fmt.Println(string(res))
	var e interface{}
	err = json.Unmarshal(res, &e)

	if err != nil {
		fmt.Println("something went wrong when unmarshalling the json")
		fmt.Println(err)
	}
	info := e.(map[string]interface{})

	return info
}