package main

import (
	"context"
	"net/http"
	"net/url"

	"github.com/go-fed/activity/pub"
	"github.com/go-fed/activity/streams/vocab"
	"github.com/go-fed/httpsig"

	"github.com/gologme/log"

	// "log"
)

var _ pub.CommonBehavior = &commonBehavior{}

type commonBehavior struct {
	db *database
	parent *Actor
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
	iri, err := url.Parse(baseURL + a.parent.name + "/outbox")
	if err != nil{
		log.Info("something went wrong with the parsing of the outbox url")
		log.Info(err)
		return
	}
	ocp, err = a.db.GetOutbox(c, iri)

	return
}

func (a *commonBehavior) NewTransport(c context.Context, actorBoxIRI *url.URL, gofedAgent string) (t pub.Transport, err error) {
	clock, err := newClock("Europe/Athens")
	if err != nil {
		log.Info("something is wrong with the clock")
		log.Info(err)
	}

	client := &http.Client{}

	getSigner, _, err := httpsig.NewSigner( []httpsig.Algorithm{httpsig.RSA_SHA256}, []string{"(request-target)", "date", "host", "digest"}, httpsig.Signature )
	postSigner, _, err := httpsig.NewSigner( []httpsig.Algorithm{httpsig.RSA_SHA256}, []string{"(request-target)", "date", "host", "digest"}, httpsig.Signature )
	if err != nil{
		log.Info("something is wrong with the httpsigner function call")
		log.Info(err)
	}

	t = pub.NewHttpSigTransport(client, "pherephone", clock, getSigner, postSigner, baseURL + a.parent.name + "#main-key", a.parent.privateKey)

	return
}
