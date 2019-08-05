# Notes on the development of `pherephone`

1. I get a `missing header "host"` error. I cannot see a way to overcome this unless I modify the `activity` library to add a "host" header.
1. ActivityPub does not specify whether a **follow** activity requires a **To** property. [ActivityPub-Example](https://github.com/tOkeshu/activitypub-example) says it doesn't. However [go-fed's activity](https://github.com/go-fed/activity) doesn't parse the `object` property to determine recipients in [sideEffectActor.prepare()](side_effect_actor.go#622) #pherephoneDev
1. When mastodon tries to follow a writefreely blog it does `"POST /api/collections/qwazix/inbox" 200 660.04181ms "http.rb/3.3.0 (Mastodon/2.8.3-cybre; +https://cybre.space/)"`