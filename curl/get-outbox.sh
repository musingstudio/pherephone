#!/bin/bash

## Get the outbox of an actor

curl https://floorb.qwazix.com/myAwesomeList1/outbox -X GET -H "Content-Type: application/ld+json; profile=\"https://www.w3.org/ns/activitystreams\"" -H "Accept: application/activity+json"