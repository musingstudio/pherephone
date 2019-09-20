package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/gologme/log"

	"github.com/writeas/activityserve"
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

	// create a logger with levels but without prefixes for easier to read
	// debug output
	printer := log.New(os.Stdout, "", 0)
	printer.EnableLevel("error")

	if *debugFlag == true {
		fmt.Println()
		fmt.Println("debug mode on")
		log.EnableLevel("info")
		printer.EnableLevel("info")
	}

	configurationFile := activityserve.Setup("config.ini", *debugFlag)

	announceReplies, _ := configurationFile.Section("general").Key("announce_replies").Bool()

	// get the list of local actors to host and the list
	// of remote actors any of them relays (boosts, announces)
	jsonFile, err := os.Open("actors.json")
	if err != nil {
		log.Info("something is wrong with the json file containing the actors")
		log.Info(err)
	}

	// Unmarshall it into a map of string arrays
	whoFollowsWho := make(map[string]map[string]interface{})
	byteValue, _ := ioutil.ReadAll(jsonFile)
	err = json.Unmarshal(byteValue, &whoFollowsWho)
	if err != nil {
		printer.Error("There's an error in your actors.json. Please check!")
		printer.Error("")
		return
	}

	actors := make(map[string]activityserve.Actor)

	// create all local actors if they don't exist yet
	for follower, data := range whoFollowsWho {
		followees := data["follow"].([]interface{})
		printer.Info()
		log.Info("Local Actor: " + follower)
		if strings.ContainsAny(follower, " \\/:*?\"<>|") {
			log.Warn("local actors can't have spaces or any of these characters in their name: \\/:*?\"<>|")
			log.Warn("Actor " + follower + " will be ignored")
			continue
		}
		localActor, err := activityserve.GetActor(follower, data["summary"].(string), "Service")
		if err != nil {
			log.Info("error creating local actor")
			return
		}
		// Now follow each one of it's users
		log.Info("Users to relay:")
		for _, followee := range followees {
			log.Info(followee)
			go localActor.Follow(followee.(string))
		}
		// Iterate over the current following users and if anybody doesn't exist
		// in the users to follow list unfollow them
		for following := range localActor.Following() {
			exists := false
			for _, followee := range followees {
				if followee.(string) == following {
					exists = true
					break
				}
			}
			if exists == false {
				go localActor.Unfollow(following)
			}
		}

		// boost everything that comes in
		localActor.OnReceiveContent = func(activity map[string]interface{}) {
			// check if we are following the person that sent us the
			// message otherwise we're open in spraying spam from whoever
			// messages us to our followers
			if _, ok := localActor.Following()[activity["actor"].(string)]; ok {
				object := activity["object"].(map[string]interface{})
				inReplyTo, ok := object["inReplyTo"]
				isReply := false
				// if the field exists and is not null and is not empty
				// then it's a reply
				if ok && inReplyTo != nil && inReplyTo != "" {
					isReply = true
				}
				// if it's a reply and announce_replies config option
				// is set to false then bail out
				if announceReplies == true || isReply == false {
					content := activity["object"].(map[string]interface{})
					go localActor.Announce(content["id"].(string))
				}
			}
		}
		actors[localActor.Name] = localActor
	}

	activityserve.Serve(actors)
}
