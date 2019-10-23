# Pherephone

A _Pherephonon_ in Greek is someone who doesn't have his own voice and just repeats things other said. _Pherephone_ is an ActivityPub relay. You set it up to follow a few accounts and it Announces everything they post.

## How to run 

Download the binary, edit `config.ini` and `actors.json` to your liking and run it.
For the time being only the ActivityPub actor endpoint is supported. Usually the url of the user profile page will work. If it doesn't you will have to find it out using **webfinger** yourself. Here is an example on how to find my actor endpoint on _fosstodon_.

[https://fosstodon.org/.well-known/webfinger?resource=acct:qwazix@fosstodon.org](https://fosstodon.org/.well-known/webfinger?resource=acct:qwazix@fosstodon.org)

There's a `-debug` flag if you want more verbose output. 

### Actors configuration

``` json
{
    "writefreelyAndFriends" : {
        "summary": "a list of favorite writers",
        "follow": [
            "https://write.as/api/collections/blog",
            "https://writing.exchange/users/write_as"
        ]
    },
    "all_about_qwazix" : {
        "summary": "wanna stalk me?",
        "follow": [
            "https://pixelfed.social/users/qwazix",
            "https://print3d.social/users/qwazix",
            "https://mixt.qwazix.com/api/collections/qwazix",
            "https://fosstodon.org/api/collections/qwazix"
        ]
    }
}
```

_Pherephone_ will create the accounts **writefreelyAndFriends@example.com** and **all_about_qwazix@example.com** and follow the users listed under each one. If you want to unfollow someone just remove any entry. Unfortunately `json` doesn't support comments so you'll have to delete it altogether. Mind the commas (there's no comma after the last entry)

### Web server configuration and https

You will probably want to run it behind a reverse proxy with a *Let's Encrypt* certificate.

Here's the configuration for *apache*. You can find similar *nginx* configuration in the [writefreely documentation](https://writefreely.org/start#production)

```
<VirtualHost *:443>
    ServerAdmin q@qzx.gr
    ProxyRequests off
    DocumentRoot /var/www
    ProxyPreserveHost On

    ServerName pherephone.example.com

    ErrorLog /var/log/httpd/error.log
    CustomLog /var/log/httpd/access.log combined

    SSLEngine on
	SSLCertificateFile "/etc/letsencrypt/live/pherephone.example.com/fullchain.pem"
	SSLCertificateKeyFile "/etc/letsencrypt/live/pherephone.example.com/privkey.pem"

    # Possible values include: debug, info, notice, warn, error, crit,
    # alert, emerg.
    LogLevel error

    <Location />
        ProxyPass http://localhost:8081/
        ProxyPassReverse http://localhost:8081/
        Order allow,deny
        Allow from all
    </Location>
</VirtualHost>
```

## General Configuration

There are four configuration values in _Pherephone_

``` ini
[general]

baseURL = https://example.com
storage = storage ; can be relative or absolute path
userAgent = "pherephone"
announce_replies = false ; whether to boost replies of followers by default
```

The `baseURL` which is, erm, self-explanatory. Set it to your (sub)domain.
`storage` which is the path where pherephone will save its data. Pherephone only uses json files in a directory structure to save its data.
`userAgent` just sets the user agent string reported by the software
`announce_replies` controls whether pherephone will boost everything the actors it follows post or only original posts.