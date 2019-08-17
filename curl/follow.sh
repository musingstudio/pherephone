#!/bin/bash

## follow our user

curl http://floorb.qwazix.com/nuerList/inbox -X POST -H "Content-Type: application/activity+json; profile=\"https://www.w3.org/ns/activitystreams\"" -H "Accept: application/activity+json" -d '{"@context":["https://www.w3.org/ns/activitystreams"],"actor":"http://writefreely.xps/api/collections/qwazix","object":"http://writefreely.xps/api/collections/qwazix","to":"http://writefreely.xps/api/collections/qwazix","type":"Follow", "id":"http://writefreely.xps/thisisfake"}'
