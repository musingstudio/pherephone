# Notes on the development of `pherephone`

1. I get a `missing header "host"` error. I cannot see a way to overcome this unless I modify the `activity` library to add a "host" header.
1. ActivityPub does not specify whether a **follow** activity requires a **To** property. [ActivityPub-Example](https://github.com/tOkeshu/activitypub-example) says it doesn't. However [go-fed's activity](https://github.com/go-fed/activity) doesn't parse the `object` property to determine recipients in [sideEffectActor.prepare()](side_effect_actor.go#622) #pherephoneDev
1. When mastodon tries to follow a writefreely blog it does `"POST /api/collections/qwazix/inbox" 200 660.04181ms "http.rb/3.3.0 (Mastodon/2.8.3-cybre; +https://cybre.space/)"`

## I don't know why I thought this

1. Go ignores `/etc/hosts` so :facepalm: (No it doesn't)

## http requests need to be sent in goroutines if you want to be able to handle incoming connections before you get a response.

Go was keeping sending me connections-refused when writefreely attempted to ping pherephone back. I thought maybe ssl, ports, forwarding. I attempted several configuration but the breakthrough came when I set up several virtual hosts on local apache and voila! Apache responded with a 503! That means that my networking setup was correct and pherephone was actually refusing requests. This happened because writefreely pinged back *before* responding to the initial request, and pherephone couldn't handle both at the same time. I tested this by setting the inbox to be another instance of pherephone on another port and yeah, the follow went through. Now I have to see how to allow golang to receive requests while sending other requests. I thought golang http requests were asynchronous anyway. (They weren't).

## I discovered a bug in writefreely

If I try to follow when I have already followed writefreely crashes.

## Thoughts about reusability of this code

The plan is to make "library" that makes it as easy as
    
        actor := MakeActor("My Name", "I'm a bot", "http://bot.farm/mybot/")
        actor.follow("https://mixt.qwazix.com/api/collections/qwazix")
        actor.post("Note", "Toot! Toot!")
    
     to create federated services

## Pherephone design

Pherephone should take a file like this 

        [
            { "favoriteArtists" : 
                [
                    "http://mastodon.social/users/favoriteArtist1",
                    "http://artalley.porn/users/favoriteArtist2",
                    "http://mastodon.art/users/favoriteArtist3"
                ]
            },
            { "famousWriters" : 
                [
                    "http://mastodon.social/users/favoriteArtist1",
                    "http://artalley.porn/users/favoriteArtist2",
                    "http://mastodon.art/users/favoriteArtist3"
                ]
            },
            { "cringeyCEOh's" : 
                [
                    "http://corporate.hell/users/brad",
                    "http://computer.money/users/mitch",
                    "http://evil.overlord/users/steven"
                ]
            },

        ]

and create a local actor with usernames `favoriteArtists`, `famousWriters` etc. 
use them to follow the listed users and boost everything they post. That is all.

## How does a mastodon boost look like in activitypub?

First, let's get my mastodon actor object

    curl -X GET https://mastodon.social/@qwazix -H "Accept: application/activity+json; profile=\"https://www.w3.org/ns/activitystreams\"" | python -m json.tool

This 302's to https://mastodon.social/users/qwazix btw. The `json.tool` makes sure the result is pretty printed in your terminal. Mastodon promptly responds with:

``` json
  "@context": [
        "https://www.w3.org/ns/activitystreams",
        "https://w3id.org/security/v1",
        {
            "manuallyApprovesFollowers": "as:manuallyApprovesFollowers",
            "toot": "http://joinmastodon.org/ns#",
            "featured": {
                "@id": "toot:featured",
                "@type": "@id"
            },
            "alsoKnownAs": {
                "@id": "as:alsoKnownAs",
                "@type": "@id"
            },
            "movedTo": {
                "@id": "as:movedTo",
                "@type": "@id"
            },
            "schema": "http://schema.org#",
            "PropertyValue": "schema:PropertyValue",
            "value": "schema:value",
            "Hashtag": "as:Hashtag",
            "Emoji": "toot:Emoji",
            "IdentityProof": "toot:IdentityProof",
            "focalPoint": {
                "@container": "@list",
                "@id": "toot:focalPoint"
            }
        }
    ],
    "id": "https://mastodon.social/users/qwazix",
    "type": "Person",
    "following": "https://mastodon.social/users/qwazix/following",
    "followers": "https://mastodon.social/users/qwazix/followers",
    "inbox": "https://mastodon.social/users/qwazix/inbox",
    "outbox": "https://mastodon.social/users/qwazix/outbox",
    "featured": "https://mastodon.social/users/qwazix/collections/featured",
    "preferredUsername": "qwazix",
    "name": "qwazix",
    "summary": "<p>I do <a href=\"https://mastodon.social/tags/photography\" class=\"mention hashtag\" rel=\"tag\">#<span>photography</span></a>, <a href=\"https://mastodon.social/tags/3D\" class=\"mention hashtag\" rel=\"tag\">#<span>3D</span></a> and other visual arts. I also code, read books and watch movies. I have built <a href=\"https://mastodon.social/tags/Rosalind\" class=\"mention hashtag\" rel=\"tag\">#<span>Rosalind</span></a> and I love <a href=\"https://mastodon.social/tags/3Dprinting\" class=\"mention hashtag\" rel=\"tag\">#<span>3Dprinting</span></a>. I also internet.</p>",
    "url": "https://mastodon.social/@qwazix",
    "manuallyApprovesFollowers": false,
    "movedTo": "https://cybre.space/users/qwazix",
    "publicKey": {
        "id": "https://mastodon.social/users/qwazix#main-key",
        "owner": "https://mastodon.social/users/qwazix",
        "publicKeyPem": "-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAwwvRyYy87zZIG/Gc87a9\nuazb6WL+tTACih+mbCLMSC/uUMgp3R7fUvRQ+KF9b2ruailcyRhQ6vPoosAQKyEA\n2rxqP5JM9+tqk4uu9TNmjmzRUG/fT1iwGTo2jgWVn8Fef/eYDH+OsTiJXDQDCwQL\nAygIyMqUikGtwfMejBxLmvscKCpn99Iz4xD7XENjmXBu7V/XZifujaymVzAGHB2l\nobhgdPxc5wHNOGY/M9/ESWslbL9d1C3xVE6S+uhP9qrGRIoHkylYPfTb6CNoTpkQ\nYfuwhtWe4HczHop6Vk89mwaSMB8khVXJS5ymoVoVpiu72JEkt6GB7za/pY8wxBky\nDwIDAQAB\n-----END PUBLIC KEY-----\n"
    },
    "tag": [
        {
            "type": "Hashtag",
            "href": "https://mastodon.social/explore/3d",
            "name": "#3d"
        },
        {
            "type": "Hashtag",
            "href": "https://mastodon.social/explore/photography",
            "name": "#photography"
        },
        {
            "type": "Hashtag",
            "href": "https://mastodon.social/explore/3dprinting",
            "name": "#3dprinting"
        },
        {
            "type": "Hashtag",
            "href": "https://mastodon.social/explore/rosalind",
            "name": "#rosalind"
        }
    ],
    "attachment": [
        {
            "type": "PropertyValue",
            "name": "website",
            "value": "<a href=\"https://qwazix.com\" rel=\"me nofollow noopener\" target=\"_blank\"><span class=\"invisible\">https://</span><span class=\"\">qwazix.com</span><span class=\"invisible\"></span></a>"
        },
        {
            "type": "PropertyValue",
            "name": "rosalind",
            "value": "<a href=\"https://rosalind.xyz\" rel=\"me nofollow noopener\" target=\"_blank\"><span class=\"invisible\">https://</span><span class=\"\">rosalind.xyz</span><span class=\"invisible\"></span></a>"
        },
        {
            "type": "PropertyValue",
            "name": "pixelfed",
            "value": "<a href=\"https://pixelfed.social/qwazix\" rel=\"me nofollow noopener\" target=\"_blank\"><span class=\"invisible\">https://</span><span class=\"\">pixelfed.social/qwazix</span><span class=\"invisible\"></span></a>"
        }
    ],
    "endpoints": {
        "sharedInbox": "https://mastodon.social/inbox"
    },
    "icon": {
        "type": "Image",
        "mediaType": "image/jpeg",
        "url": "https://files.mastodon.social/accounts/avatars/000/284/505/original/ca72250e2d1aa78d.jpg"
    },
    "image": {
        "type": "Image",
        "mediaType": "image/png",
        "url": "https://files.mastodon.social/accounts/headers/000/284/505/original/K4M0ZD6AOKOC.png"
    }
}
```

As you can see we can read the outbox using `https://mastodon.social/users/qwazix/outbox`. Mastodon replies quite helfully that we need to append `?page=true` to fetch the first page of toots.

``` json
{
    "@context": "https://www.w3.org/ns/activitystreams",
    "id": "https://mastodon.social/users/qwazix/outbox",
    "type": "OrderedCollection",
    "totalItems": 7528,
    "first": "https://mastodon.social/users/qwazix/outbox?page=true",
    "last": "https://mastodon.social/users/qwazix/outbox?min_id=0&page=true"
}
```

So we can now fetch our toots, nicely printed with this command:

    curl https://mastodon.social/users/qwazix/outbox?page=true -H "Content-Type: application/ld+json; profile=\"https://www.w3.org/ns/activitystreams\"" | python -m json.tool

Here's what a boost looks like

``` json
{
    "id": "https://mastodon.social/users/qwazix/statuses/102565077202414254/activity",
    "type": "Announce",
    "actor": "https://mastodon.social/users/qwazix",
    "published": "2019-08-05T15:27:58Z",
    "to": [
        "https://www.w3.org/ns/activitystreams#Public"
    ],
    "cc": [
        "https://cybre.space/users/tzo",
        "https://mastodon.social/users/qwazix/followers"
    ],
    "object": "https://cybre.space/users/tzo/statuses/102564367759300737",
    "atomUri": "https://mastodon.social/users/qwazix/statuses/102565077202414254/activity"
}
```