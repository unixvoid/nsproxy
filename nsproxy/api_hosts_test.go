package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestApiHostsHandler(t *testing.T) {
	readConf()
	redisClient, _ := initRedisConnection()

	// add a test host to use
	TestClusterHandler(t)
	// sleep so the async task can finish
	time.Sleep(1 * time.Millisecond)

	getData := url.Values{}

	r, _ := http.NewRequest("GET", "", strings.NewReader(getData.Encode()))
	w := httptest.NewRecorder()

	apiHostsHandler(w, r, redisClient)
	if w.Code == 200 {
		// we expect 200
		t.Log("\x1b[31mSending GET :: /hosts\x1b[39m")
		t.Log("Recieved 200 correctly")
		t.Logf("Body returned: \x1b[36m%s\x1b[39m", w.Body)
	} else {
		// error
		t.Errorf("Expected 200, got %v instead", w.Code)
		t.Errorf("Body returned: %s", w.Body)
	}
	// teardown test entries in redis
	TestTeardown(t)
}
