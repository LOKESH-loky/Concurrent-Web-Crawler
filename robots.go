package main

import (
	"net/http"
	"net/url"
	"sync"

	"github.com/temoto/robotstxt"
)

var robotsMap = make(map[string]*robotstxt.RobotsData)
var robotsMutex sync.Mutex

func canFetch(urlStr string) bool {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return false
	}
	baseURL := parsedURL.Scheme + "://" + parsedURL.Host

	robotsMutex.Lock()
	robots, exists := robotsMap[baseURL]
	robotsMutex.Unlock()

	if !exists {
		resp, err := http.Get(baseURL + "/robots.txt")
		if err != nil {
			return true
		}
		defer resp.Body.Close()
		robotsData, err := robotstxt.FromResponse(resp)
		if err != nil {
			return true
		}
		robotsMutex.Lock()
		robotsMap[baseURL] = robotsData
		robots = robotsData
		robotsMutex.Unlock()
	}

	uaGroup := robots.FindGroup("YourUserAgent")
	return uaGroup.Test(parsedURL.Path)
}
