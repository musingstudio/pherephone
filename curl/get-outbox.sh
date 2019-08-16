#!/bin/bash

## Get the outbox of an actor

curl http://localhost:8081/nuerList/outbox -X GET -H "Content-Type: application/ld+json; profile=\"https://www.w3.org/ns/activitystreams\"" -H "Accept: application/activity+json"