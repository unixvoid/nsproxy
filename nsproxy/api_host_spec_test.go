package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestApiHostSpecHandler(t *testing.T) {
	readConf()
	redisClient, _ := initRedisConnection()
	// add a test host to use
	TestClusterHandler(t)

	postData := url.Values{}
	postData.Set("cluster", "testcluster")
	postData.Add("host", "test.domain")

	r, _ := http.NewRequest("POST", "", strings.NewReader(postData.Encode()))
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	apiHostSpecHandler(w, r, redisClient)
	if w.Code == 200 {
		// we expect 200
		t.Log("\x1b[31mSending POST :: /hostspec\x1b[39m")
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
