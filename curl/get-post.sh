#!/bin/bash

## Get the outbox of an actor

curl http://floorb.qwazix.com/nuerList/post/1tAhKpAxFtoWzMXj -X GET -H "Content-Type: application/ld+json; profile=\"https://www.w3.org/ns/activitystreams\"" -H "Accept: application/activity+json"