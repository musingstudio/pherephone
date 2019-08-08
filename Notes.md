# Notes on the development of `pherephone`

1. I get a `missing header "host"` error. I cannot see a way to overcome this unless I modify the `activity` library to add a "host" header.
1. ActivityPub does not specify whether a **follow** activity requires a **To** property. [ActivityPub-Example](https://github.com/tOkeshu/activitypub-example) says it doesn't. However [go-fed's activity](https://github.com/go-fed/activity) doesn't parse the `object` property to determine recipients in [sideEffectActor.prepare()](side_effect_actor.go#622) #pherephoneDev
1. When mastodon tries to follow a writefreely blog it does `"POST /api/collections/qwazix/inbox" 200 660.04181ms "http.rb/3.3.0 (Mastodon/2.8.3-cybre; +https://cybre.space/)"`
1. Go ignores `/etc/hosts` so :facepalm: (No it doesn't)
1. Go was keeping sending me connections-refused when writefreely attempted to ping pherephone back. I thought maybe ssl, ports, forwarding. I attempted several configuration but the breakthrough came when I set up several virtual hosts on local apache and voila! Apache responded with a 503! That means that my networking setup was correct and pherephone was actually refusing requests. I tested this by setting the inbox to be another instance of pherephone on another port and yeah, the follow went through. Now I have to see how to allow golang to receive requests while sending other requests. I thought golang http requests were asynchronous anyway.

INSERT INTO "main"."remoteusers" ("id") VALUES ('1');

If I try to follow when I have already followed writefreely crashes.