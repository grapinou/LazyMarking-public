package worktool

import (
	"net/http"
	"net/http/cookiejar"
)

var Client *http.Client

func init() {
	jar, _ := cookiejar.New(nil)
	Client = &http.Client{
		Jar: jar,
	}
}
