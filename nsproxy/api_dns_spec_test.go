package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestDnsSpecHandler(t *testing.T) {
	readConf()
	redisClient, _ := initRedisConnection()
	// create dns entry to test with
	TestDnsHandler(t)

	postData := url.Values{}
	postData.Set("dnstype", "a")
	postData.Add("domain", "test.domain")

	r, _ := http.NewRequest("POST", "", strings.NewReader(postData.Encode()))
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	apiDnsSpecHandler(w, r, redisClient)
	if w.Code == 200 {
		// we expect 200
		t.Log("Recieved 200 correctly")
		t.Logf("Body returned: \x1b[36m%s\x1b[39m", w.Body)
	} else {
		// error
		t.Errorf("Expected 200, got %v instead", w.Code)
		t.Errorf("Body returned: %s", w.Body)
	}
	// remove dns entry
	TestDnsRmHandler(t)
}
