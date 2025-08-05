package main

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
