package main

import (
	"fmt"
	"net/http"
	"strings"

	"gopkg.in/redis.v3"
)

func apiClusterSpecHandler(w http.ResponseWriter, r *http.Request, redisClient *redis.Client) {
	queryCluster := strings.TrimSpace(r.FormValue("cluster"))
	hosts, err := redisClient.SInter("index:live").Result()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
	}

	for _, i := range hosts {
		// we now break at ':' and save the clusters piece
		s := strings.SplitN(i, ":", 2)
		if s[0] == queryCluster {
			// if the custer matches the query, throw the host in a tmp set
			redisClient.SAdd("tmp:cluster:index", s[1])
		}
	}
	// grab the set and delete
	clusters, err := redisClient.SInter("tmp:cluster:index").Result()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
	}
	if fmt.Sprintf("%x", clusters) == "[]" {
		// empty reply, return 400
		w.WriteHeader(http.StatusBadRequest)
	} else {
		redisClient.Del("tmp:cluster:index")
		fmt.Fprintln(w, clusters)
	}
}
