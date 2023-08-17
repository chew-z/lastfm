package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/joho/godotenv/autoload"
	"github.com/shkh/lastfm-go/lastfm"
)

type Session struct {
	Key   string `json:"key,omitempty"`
	Token string `json:"token,omitempty"`
	User  string `json:"user,omitempty"`
}

var (
	apiKey    = os.Getenv("API_KEY")
	apiSecret = os.Getenv("API_SECRET")
	api       *lastfm.Api
	uRL       = os.Getenv("URL")
	path      = os.Getenv("JSON_PATH")
	P         lastfm.P
	session   Session
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
	router.GET("/callback", callback)
	router.GET("/save", save)
	router.GET("/user", user)

	router.Run("localhost:8086")
}

func nowPlaying(c *gin.Context) {
	song := c.PostForm("song")
	split := strings.Split(song, " - ")
	artist := split[0]
	track := split[1]
	start := time.Now().Unix()

	// file := path + "playing.json"
	// trackFile, err := os.Open(file)
	// if err != nil {
	// 	log.Println("opening playing.json file", err.Error())
	// }
	// var p  lastfm.P
	// jsonParser := json.NewDecoder(trackFile)
	// if err = jsonParser.Decode(&p); err != nil {
	// 	log.Println("parsing playing.json file", err.Error())
	// }

	p := lastfm.P{"artist": artist, "track": track, "timestamp": start}
	updatedTrack, err := api.Track.UpdateNowPlaying(p)
	if err != nil {
		log.Println(err)
		return
	}
	uP := lastfm.P{"artist": updatedTrack.Artist.Name, "track": updatedTrack.Track.Name, "timestamp": start}
	log.Println("Now playing: ", uP["artist"], uP["track"])
	P = uP

	c.String(http.StatusOK, "Now playing: %s - %s", uP["artist"], uP["track"])
	return
}

func scrobble(c *gin.Context) {
	p := P
	if p != nil {
		log.Println("Now scrobbling: ", p["artist"], p["track"])
		p["chosenByUser"] = 0
	}
	sP, err := api.Track.Scrobble(p)
	if err != nil {
		log.Println(err)
		return
	}
	accepted := sP.Accepted
	if accepted == "1" {
		track := sP.Scrobbles[0].Track.Name
		artist := sP.Scrobbles[0].Artist.Name
		c.String(http.StatusOK, "Scrobbled %s - %s with result: %s", artist, track, accepted)
		return
	}
	c.String(http.StatusOK, "Scrobbled with result: %s", accepted)
	return
}

func saveNowPlaying(c *gin.Context) {
	trackJson, err := json.Marshal(&P)
	if err != nil {
		log.Println(err.Error())
	}
	file := path + "playing.json"
	err = ioutil.WriteFile(file, trackJson, 0644)
	if err != nil {
		log.Println(err.Error())
	}
	log.Println("Playing: ", P)
	c.String(http.StatusOK, "Saved now playing")
	return
}

func callback(c *gin.Context) {
	token := c.Query("token")
	api.LoginWithToken(token)
	session.Token = token
	session.Key = api.GetSessionKey()
	result, err := api.User.GetInfo(nil)
	if err != nil {
		log.Fatal(err.Error())
	}
	session.User = result.Name
	log.Println("Session: ", session)
	c.Redirect(http.StatusFound, uRL+"/save")
}

func save(c *gin.Context) {
	sessionJson, err := json.Marshal(&session)
	if err != nil {
		log.Println(err.Error())
	}
	file := path + "session.json"
	err = ioutil.WriteFile(file, sessionJson, 0644)
	if err != nil {
		log.Println(err.Error())
	}
	log.Println("Session: ", session)
	c.String(http.StatusOK, "Saved session")
	return
}

func user(c *gin.Context) {
	result, err := api.User.GetInfo(nil)
	if err != nil {
		log.Fatal(err.Error())
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
