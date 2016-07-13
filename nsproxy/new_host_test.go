package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestClusterHandler(t *testing.T) {
	readConf()
	initLogger()
	redisClient, _ := initRedisConnection()

	postData := url.Values{}
	postData.Set("hostname", "test.domain")
	postData.Add("cluster", "testcluster")
	postData.Add("ip", "127.0.0.1")
	postData.Add("port", "8080")

	r, _ := http.NewRequest("POST", "", strings.NewReader(postData.Encode()))
	// spoof remote address
	r.RemoteAddr = "127.0.0.1"

	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	clusterHandler(w, r, redisClient)
	if w.Code == 200 {
		// we expect 200
		t.Log("Test host created successfully")
	} else {
		// error
		t.Errorf("Expected 200, got %v instead", w.Code)
		t.Errorf("Body returned: %s", w.Body)
	}
}

func TestTeardown(t *testing.T) {
	readConf()
	redisClient, _ := initRedisConnection()

	redisClient.Del("cweight:testcluster:test.domain")
	redisClient.Del("index:cluster:testcluster")
	redisClient.Del("index:live")
	redisClient.Del("index:master")
	redisClient.Del("list:cluster:testcluster")
	redisClient.Del("port:testcluster:test.domain")
	redisClient.Del("cluster:testcluster:test.domain")
	redisClient.Del("weight:testcluster:test.domain")
	redisClient.Del("state:cluster:testcluster")
	redisClient.Del("dns:a:test.domain.")
	redisClient.Del("index:dns")
}
