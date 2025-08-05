package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shkh/lastfm-go/lastfm"
)

func nowPlaying(c *gin.Context) {
	song := c.PostForm("song")
	album := c.DefaultPostForm("album", "noalbum")
	sP := c.PostForm("start")
	start, err := strconv.ParseInt(sP, 10, 64)
	if err != nil {
		start = time.Now().Unix()
	}
	split := strings.Split(song, " - ")
	if len(split) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid song format. Expected 'artist - track'"})
		return
	}
	artist := split[0]
	track := split[1]
	if x, found := kaszka.Get("nowPlaying"); found {
		s := x.(*Scrobble)
		cacheExpiration, err := strconv.Atoi(os.Getenv("CACHE_EXPIRATION_MS"))
		if err != nil {
			cacheExpiration = 180000
		}
		// Same song within 3 minutes - ignore
		if (start-s.Time) < int64(cacheExpiration) && song == s.Song { // 3 minutes * 60 secodnds * 10000 miliseconds
			c.JSON(http.StatusOK, gin.H{"message": song + " is already playing"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
	c.JSON(http.StatusOK, gin.H{"message": "Now playing: " + uP.Artist + " - " + uP.Title + " [" + uP.Album + "]"})
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
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			accepted := sP.Accepted
			if accepted == "1" {
				s.Scrobbled = true
				kaszka.SetDefault("nowPlaying", s)
				track := sP.Scrobbles[0].Track.Name
				artist := sP.Scrobbles[0].Artist.Name
				c.JSON(http.StatusOK, gin.H{"message": "Scrobbled " + artist + " - " + track + " with result: " + accepted})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "Scrobbled " + s.Song + " with result: " + accepted})
			return
		} else {
			c.JSON(http.StatusOK, gin.H{"message": "Scrobbled " + s.Song + " already"})
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "Seems like cache is empty"})
	return
}

func saveNowPlaying(c *gin.Context) {
	if x, found := kaszka.Get("nowPlaying"); found {
		s := x.(*Scrobble)
		P := lastfm.P{"artist": s.Artist, "track": s.Title, "timestamp": s.Time, "chosenByUser": 0}
		trackJson, err := json.Marshal(&P)
		if err != nil {
			log.Println(err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		file := path + "nowPlaying.json"
		err = os.WriteFile(file, trackJson, 0666)
		if err != nil {
			log.Println(err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		log.Println("Playing: ", P)
		c.JSON(http.StatusOK, gin.H{"message": "Saved now playing"})
		return
	} else {
		c.JSON(http.StatusNotFound, gin.H{"error": "Could not find now playing in cache"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	session.User = result.Name
	log.Println("Session: ", session)
	c.Redirect(http.StatusFound, uRL+"/saveSession")
}

func saveSession(c *gin.Context) {
	sessionJson, err := json.Marshal(&session)
	if err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	file := path + "session.json"
	err = os.WriteFile(file, sessionJson, 0666)
	if err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	log.Println("Session: ", session)
	c.JSON(http.StatusOK, gin.H{"message": "Saved session"})
	return
}

func displayUser(c *gin.Context) {
	result, err := api.User.GetInfo(nil)
	if err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
