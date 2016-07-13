package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestDnsEntriesHandler(t *testing.T) {
	readConf()
	redisClient, _ := initRedisConnection()
	// create dns entry to test with
	TestDnsHandler(t)

	getData := url.Values{}

	r, _ := http.NewRequest("GET", "", strings.NewReader(getData.Encode()))
	w := httptest.NewRecorder()

	dnsHostsHandler(w, r, redisClient)
	if w.Code == 200 {
		// we expect 200
		t.Log("\x1b[31mSending GET :: /dns\x1b[39m")
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
