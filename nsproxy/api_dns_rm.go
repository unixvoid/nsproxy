package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/unixvoid/nsproxy/pkg/nslog"
	"gopkg.in/redis.v3"
)

func dnsRmHandler(w http.ResponseWriter, r *http.Request, redisClient *redis.Client) {
	r.ParseForm()

	rmType := strings.ToLower(strings.TrimSpace(r.FormValue("dnstype")))
	rmDomain := strings.TrimSpace(r.FormValue("domain"))

	if len(rmDomain) == 0 {
		nslog.Debug.Println("domain not set, exiting..")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// fully qualify domain if not done already
	if string(rmDomain[len(rmDomain)-1]) != "." {
		rmDomain = fmt.Sprintf("%s.", rmDomain)
	}

	if len(rmType) == 0 {
		// if type not set, nix them all
		nslog.Debug.Printf("removing all dns types for %s", rmDomain)
		redisClient.Del(fmt.Sprintf("dns:a:%s", rmDomain))
		redisClient.Del(fmt.Sprintf("dns:aaaa:%s", rmDomain))
		redisClient.Del(fmt.Sprintf("dns:cname:%s", rmDomain))

		// remove dns entry from the custom list
		redisClient.SRem("index:dns", fmt.Sprintf("a:%s", rmDomain))
		redisClient.SRem("index:dns", fmt.Sprintf("aaaa:%s", rmDomain))
		redisClient.SRem("index:dns", fmt.Sprintf("cname:%s", rmDomain))
	} else {
		// just remove the specific type
		nslog.Debug.Printf("removing %s entry for %s", rmType, rmDomain)
		redisClient.Del(fmt.Sprintf("dns:%s:%s", rmType, rmDomain))

		// remove dns entry from the custom list
		redisClient.SRem("index:dns", fmt.Sprintf("%s:%s", rmType, rmDomain))
	}
	w.WriteHeader(http.StatusOK)
}
