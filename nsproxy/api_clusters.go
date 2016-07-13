package main

import (
	"fmt"
	"net/http"
	"strings"

	"gopkg.in/redis.v3"
)

func apiClustersHandler(w http.ResponseWriter, r *http.Request, redisClient *redis.Client) {
	hosts, err := redisClient.SInter("index:live").Result()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
	}
	for _, i := range hosts {
		// we now break at ':' and save the clusters piece
		s := strings.SplitN(i, ":", 2)
		// toss them all into a tmp redis set
		redisClient.SAdd("tmp:cluster:index", s[0])
	}
	// grab the set and delete
	clusters, err := redisClient.SInter("tmp:cluster:index").Result()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
	}
	redisClient.Del("tmp:cluster:index")
	fmt.Fprintln(w, clusters)
}
