package main

import (
	"io/ioutil"
	"net/http"
	"time"
)

func fetch(url string) (string, error) {
	var resp *http.Response
	var err error
	for i := 0; i < 3; i++ {
		resp, err = http.Get(url)
		if err == nil {
			break
		}
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(bodyBytes), nil
}
