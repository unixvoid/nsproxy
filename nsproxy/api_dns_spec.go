package main

import (
	"fmt"
	"net/http"
	"strings"

	"gopkg.in/redis.v3"
)

func apiDnsSpecHandler(w http.ResponseWriter, r *http.Request, redisClient *redis.Client) {
	dnsType := strings.TrimSpace(r.FormValue("dnstype"))
	domainValue := strings.TrimSpace(r.FormValue("domain"))

	// fully qualify if not already
	if string(domainValue[len(domainValue)-1]) != "." {
		domainValue = fmt.Sprintf("%s.", domainValue)
	}

	// default to a record
	if len(dnsType) == 0 {
		dnsType = "a"
	}

	ip, err := redisClient.Get(fmt.Sprintf("dns:%s:%s", dnsType, domainValue)).Result()
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
	}

	fmt.Fprintln(w, ip)
}
