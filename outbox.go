package main

import (
	// "log"
	"github.com/gologme/log"

	"os"
)

type outbox struct{
	posts map[int]map[string]string
	path string
}

// makeOutbox creates an outbox, and its respective file on disk
func makeOutbox(actor Actor) error {
	log.Info("Creating outbox for "+ actor.name)
	// the actor ought to exist otherwise something is really wrong
	_, err := os.Stat("actors" + slash + actor.name)
	if err != nil {
		log.Info("Can't access actor diretory, something is wrong")
		return err
	}


	return nil
}