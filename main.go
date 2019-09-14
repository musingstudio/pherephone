package main

import (
	"flag"
	"fmt"

	// "os"
	// "strings"

	// "errors"

	// "encoding/json"
	// "io/ioutil"
	// "net/http"

	// "net/url"
	// "context"
	// "html"

	"github.com/gologme/log"

	// "github.com/go-fed/activity/streams"
	// "github.com/gorilla/mux"
	// "gopkg.in/ini.v1"
	// "github.com/davecgh/go-spew/spew"

	"../activityserve"
)

var err error

func main() {

	// This is here for debugging purposes. I want to be able to easily spot in the terminal
	// when a single execution starts
	fmt.Println()
	fmt.Println("======================= PHeRePHoNe ==========================")

	// introduce ourselves
	fmt.Println()
	fmt.Println("Pherephone follows some accounts and boosts")
	fmt.Println("whatever they post to our followers. See config.ini ")
	fmt.Println("for more information and how to set up. ")

	debugFlag := flag.Bool("debug", false, "set to true to get debugging information in the console")
	flag.Parse()

	if *debugFlag == true {
		log.EnableLevel("info")
		log.EnableLevel("error")
	}

	activityserve.Setup("config.ini", *debugFlag)

	// actor, _ := activityserve.MakeActor("activityserve_test_actor_2", "This is an activityserve test actor", "Service")
	// actor, _ := activityserve.LoadActor("activityserve_test_actor_2")
	// actor.Follow("https://cybre.space/users/tzo")
	// actor.CreateNote("Hello World!")

	actor, _ :=
	activityserve.LoadActor("activityserve_test_actor_2")
	actor.CreateNote("I'm building #ActivityPub stuff", "")

	activityserve.Serve()
}
