package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestDnsHandler(t *testing.T) {
	readConf()
	redisClient, _ := initRedisConnection()
	postData := url.Values{}

	postData.Set("dnstype", "a")
	postData.Add("domain", "test.domain")
	postData.Add("value", "127.0.0.1")

	r, _ := http.NewRequest("POST", "", strings.NewReader(postData.Encode()))
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	dnsHandler(w, r, redisClient)
	if w.Code == 200 {
		// we expect 200
		t.Log("\x1b[31mSending POST :: /dns\x1b[39m")
		t.Log("Recieved 200 correctly")
	} else {
		// error
		t.Errorf("Expected 200, got %v instead", w.Code)
	}
}
