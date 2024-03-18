package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/joho/godotenv/autoload"
	"github.com/patrickmn/go-cache"
	"github.com/shkh/lastfm-go/lastfm"
)

type Session struct {
	Key   string `json:"key,omitempty"`
	Token string `json:"token,omitempty"`
	User  string `json:"user,omitempty"`
}

type Scrobble struct {
	Song      string
	Album     string
	Artist    string
	Title     string
	Time      int64
	Scrobbled bool
}

var (
	apiKey    = os.Getenv("API_KEY")
	apiSecret = os.Getenv("API_SECRET")
	api       *lastfm.Api
	uRL       = os.Getenv("URL")
	path      = os.Getenv("JSON_PATH")
	session   Session
	kaszka    = cache.New(24*time.Hour, 30*time.Minute)
)

func init() {
	api = lastfm.New(apiKey, apiSecret)
	file := path + "session.json"
	sessionFile, err := os.Open(file)
	if err != nil {
		log.Println("opening session.json file", err.Error())
	}

	jsonParser := json.NewDecoder(sessionFile)
	if err = jsonParser.Decode(&session); err != nil {
		log.Println("parsing session.json file", err.Error())
	}
	// log.Println(session.Key)
	api.SetSession(session.Key)
	kaszka.Flush()
}

func main() {
	router := gin.Default()
	router.LoadHTMLGlob("templates/*")
	router.GET("/", func(c *gin.Context) {
		u := api.GetAuthRequestUrl(uRL + "/callback")
		c.HTML(http.StatusOK, "main.html", gin.H{
			"URL": u,
		})
	})

	router.POST("/nowplaying", nowPlaying)
	router.GET("/scrobble", scrobble)
	router.GET("/saveNowPlaying", saveNowPlaying)
	router.GET("/saveSession", saveSession)
	router.GET("/displayUser", displayUser)
	router.GET("/callback", callback)

	router.Run("localhost:8086")
}

func nowPlaying(c *gin.Context) {
	song := c.PostForm("song")
	album := c.DefaultPostForm("album", "noalbum")
	sP := c.PostForm("start")
	start, err := strconv.ParseInt(sP, 10, 64)
	if err != nil {
		start = time.Now().Unix()
	}
	split := strings.Split(song, " - ")
	artist := split[0]
	track := split[1]
	if x, found := kaszka.Get("nowPlaying"); found {
		s := x.(*Scrobble)
		// log.Println(time.Unix(start, 0), " / ", time.Unix(s.Time, 0))
		// Same song within 3 minutes - ignore
		if (start-s.Time) < 180000 && song == s.Song { // 3 minutes * 60 secodnds * 10000 miliseconds
			c.String(http.StatusOK, "%s is already playing", song)
			return
		}
	}
	p := lastfm.P{"artist": artist, "track": track, "timestamp": start}
	if album != "noalbum" && album != "" {
		p["album"] = album
	}
	// log.Println(p)
	updatedTrack, err := api.Track.UpdateNowPlaying(p)
	if err != nil {
		log.Println(err)
		c.String(http.StatusOK, err.Error())
		return
	}
	uP := &Scrobble{
		Song:      song,
		Album:     updatedTrack.Album.Name,
		Artist:    updatedTrack.Artist.Name,
		Title:     updatedTrack.Track.Name,
		Time:      start,
		Scrobbled: false,
	}
	log.Println(*uP)
	log.Println("Now playing: ", uP.Artist, uP.Title, uP.Album)
	kaszka.SetDefault("nowPlaying", uP)
	c.String(http.StatusOK, "Now playing: %s - %s [%s]", uP.Artist, uP.Title, uP.Album)
	return
}

func scrobble(c *gin.Context) {
	if x, found := kaszka.Get("nowPlaying"); found {
		s := x.(*Scrobble)
		p := lastfm.P{"album": s.Album, "artist": s.Artist, "track": s.Title, "timestamp": s.Time, "chosenByUser": 0}
		// log.Println(p)
		if s.Scrobbled == false {
			log.Println("Now scrobbling: ", p["artist"], p["track"], p["album"])
			sP, err := api.Track.Scrobble(p)
			if err != nil {
				log.Println(err)
				c.String(http.StatusOK, err.Error())
				return
			}
			accepted := sP.Accepted
			if accepted == "1" {
				s.Scrobbled = true
				kaszka.SetDefault("nowPlaying", s)
				track := sP.Scrobbles[0].Track.Name
				artist := sP.Scrobbles[0].Artist.Name
				c.String(http.StatusOK, "Scrobbled %s - %s with result: %s", artist, track, accepted)
				return
			}
			c.String(http.StatusOK, "Scrobbled %s with result: %s", s.Song, accepted)
			return
		} else {
			c.String(http.StatusOK, "Scrobbled %s already", s.Song)
			return
		}
	}
	c.String(http.StatusOK, "Seems like cache is empty")
	return
}

func saveNowPlaying(c *gin.Context) {
	if x, found := kaszka.Get("nowPlaying"); found {
		s := x.(*Scrobble)
		P := lastfm.P{"artist": s.Artist, "track": s.Title, "timestamp": s.Time, "chosenByUser": 0}
		trackJson, err := json.Marshal(&P)
		if err != nil {
			log.Println(err.Error())
			c.String(http.StatusOK, err.Error())
			return
		}
		file := path + "nowPlaying.json"
		err = ioutil.WriteFile(file, trackJson, 0666)
		if err != nil {
			log.Println(err.Error())
			c.String(http.StatusOK, err.Error())
			return
		}
		log.Println("Playing: ", P)
		c.String(http.StatusOK, "Saved now playing")
		return
	} else {
		c.String(http.StatusOK, "Could not find now playing in cache")
		return
	}
}

func callback(c *gin.Context) {
	token := c.Query("token")
	api.LoginWithToken(token)
	session.Token = token
	session.Key = api.GetSessionKey()
	result, err := api.User.GetInfo(nil)
	if err != nil {
		log.Println(err.Error())
		c.String(http.StatusOK, err.Error())
		return
	}
	session.User = result.Name
	log.Println("Session: ", session)
	c.Redirect(http.StatusFound, uRL+"/save")
}

func saveSession(c *gin.Context) {
	sessionJson, err := json.Marshal(&session)
	if err != nil {
		log.Println(err.Error())
		c.String(http.StatusOK, err.Error())
		return
	}
	file := path + "session.json"
	err = ioutil.WriteFile(file, sessionJson, 0666)
	if err != nil {
		log.Println(err.Error())
		c.String(http.StatusOK, err.Error())
		return
	}
	log.Println("Session: ", session)
	c.String(http.StatusOK, "Saved session")
	return
}

func displayUser(c *gin.Context) {
	result, err := api.User.GetInfo(nil)
	if err != nil {
		log.Println(err.Error())
		c.String(http.StatusOK, err.Error())
		return
	}
	// session.User = result.Name
	// log.Println("Session: ", session)

	c.HTML(
		http.StatusOK,
		"show.html",
		gin.H{
			// "Id":       result.Id,
			"Name":     result.Name,
			"RealName": result.RealName,
			"Url":      result.Url,
			"title":    "Show user",
		},
	)
	return
}
