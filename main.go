package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/joho/godotenv/autoload"
	"github.com/patrickmn/go-cache"
	"github.com/shkh/lastfm-go/lastfm"
)

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
	router.POST("/scrobble", scrobble)
	router.POST("/saveNowPlaying", saveNowPlaying)
	router.POST("/saveSession", saveSession)
	router.GET("/displayUser", displayUser)
	router.GET("/callback", callback)

	serverAddr := os.Getenv("SERVER_ADDR")
	if serverAddr == "" {
		serverAddr = "localhost:8086"
	}
	router.Run(serverAddr)
}

