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
	apiKey      = os.Getenv("API_KEY")
	apiSecret   = os.Getenv("API_SECRET")
	api         *lastfm.Api
	callbackURL = os.Getenv("CALLBACK_URL")
	P           lastfm.P
	session     Session
)

func init() {
	api = lastfm.New(apiKey, apiSecret)

	sessionFile, err := os.Open("session.json")
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
		u := api.GetAuthRequestUrl(callbackURL)
		c.HTML(http.StatusOK, "main.html", gin.H{
			"title": "Main page",
			"URL":   u,
		})
	})

	router.GET("/callback", callback)
	router.POST("/nowplaying", nowPlaying)
	router.GET("/scrobble", scrobble)
	router.GET("/user", user)
	router.GET("/save", save)
	router.Run(":8080") // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
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
	// sessionJson, err := json.Marshal(&session)
	// if err != nil {
	// 	log.Println(err.Error())
	// }
	// err = ioutil.WriteFile("session.json", sessionJson, 0644)
	c.Redirect(http.StatusFound, "http://localhost:8080/save")
}

func save(c *gin.Context) {
	sessionJson, err := json.Marshal(&session)
	if err != nil {
		log.Println(err.Error())
	}
	err = ioutil.WriteFile("session.json", sessionJson, 0644)
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

func nowPlaying(c *gin.Context) {
	song := c.PostForm("song")
	split := strings.Split(song, " - ")
	artist := split[0]
	track := split[1]

	start := time.Now().Unix()
	p := lastfm.P{"artist": artist, "track": track}
	p["timestamp"] = start
	_, err := api.Track.UpdateNowPlaying(p)
	if err != nil {
		log.Println(err)
		return
	}
	P = p
	c.String(http.StatusOK, "Now playing: %s - %s", p["artist"], p["track"])
	return
}
func scrobble(c *gin.Context) {
	p := P
	_, err := api.Track.Scrobble(p)
	if err != nil {
		log.Println(err)
		return
	}
	c.String(http.StatusOK, "Scrobbled %s - %s", p["artist"], p["track"])
	return
}
