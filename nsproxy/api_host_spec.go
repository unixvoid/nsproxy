package main

import (
	"fmt"
	"net/http"
	"strings"

	"gopkg.in/redis.v3"
)

func apiHostSpecHandler(w http.ResponseWriter, r *http.Request, redisClient *redis.Client) {
	cluster := strings.TrimSpace(r.FormValue("cluster"))
	host := strings.TrimSpace(r.FormValue("host"))

	ip, err := redisClient.Get(fmt.Sprintf("cluster:%s:%s", cluster, host)).Result()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
	}
	fmt.Fprintln(w, ip)
}
