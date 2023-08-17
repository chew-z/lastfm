# lastfm

```
curl --data "song=Holmes Ives - Strong Enough" -X POST http://localhost:8080/nowplaying

curl http://localhost:8080/scrobble

```

* Login first on `` http://localhost:8080/``. It stores session key in session.json so authorization has unlimited life theoretically.

* Configure parameters in .env file or use export

```
API_KEY: ""
API_SECRET: ""
URL: "https://localhost:8080"
JSON_PATH: "/home/.../../"
```

* when in doubts read the code

Put it behind some nginx on a server or on Google Cloud...

Enjoy
