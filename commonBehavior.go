package main

import (
	"context"
	"net/http"
	"net/url"

	"github.com/go-fed/activity/pub"
	"github.com/go-fed/activity/streams/vocab"
	"github.com/go-fed/httpsig"

	"crypto/rand"
	"crypto/rsa"


	"fmt"
)

var _ pub.CommonBehavior = &commonBehavior{}

type commonBehavior struct {
	db *database
}

func newCommonBehavior(db *database) *commonBehavior {
	return &commonBehavior{
		db: db,
	}
}

func (a *commonBehavior) AuthenticateGetInbox(c context.Context, w http.ResponseWriter, r *http.Request) (authenticated bool, err error) {
	// TODO
	return true, nil
}

func (a *commonBehavior) AuthenticateGetOutbox(c context.Context, w http.ResponseWriter, r *http.Request) (authenticated bool, err error) {
	// TODO
	return true, nil
}

func (a *commonBehavior) GetOutbox(c context.Context, r *http.Request) (ocp vocab.ActivityStreamsOrderedCollectionPage, err error) {
	//TODO
	// var iri *url.URL
	iri, err := url.Parse("http://floorb.qwazix.com/actor/outbox")
	if err != nil{
		fmt.Println("something went wrong with the parsing of the outbox url")
		fmt.Println(err)
		return
	}
	ocp, err = a.db.GetOutbox(c, iri)

	return
}

func (a *commonBehavior) NewTransport(c context.Context, actorBoxIRI *url.URL, gofedAgent string) (t pub.Transport, err error) {
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

	t = pub.NewHttpSigTransport(client, "pherephoneOfficial", clock, getSigner, postSigner, pubKeyId, privKey)

	return
}
