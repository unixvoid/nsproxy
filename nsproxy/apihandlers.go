package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/unixvoid/nsproxy/pkg/nslog"
	"gopkg.in/redis.v3"
)

func dnsHandler(w http.ResponseWriter, r *http.Request, redisClient *redis.Client) {
	r.ParseForm()

	dnsType := strings.ToLower(strings.TrimSpace(r.FormValue("dnstype")))
	domain := strings.TrimSpace(r.FormValue("domain"))
	domainValue := strings.TrimSpace(r.FormValue("value"))
	if len(dnsType) == 0 {
		// default to aname entry
		dnsType = "a"
	}

	if dnsType == "cname" {
		// if we are dealing with a CNAME entry fully qualify it
		if string(domainValue[len(domainValue)-1]) != "." {
			domainValue = fmt.Sprintf("%s.", domainValue)
		}
	}

	// make sure domain and value are set
	if (len(domain) == 0) || (len(domainValue) == 0) {
		nslog.Debug.Println("domain or value not set, exiting..")
		w.WriteHeader(http.StatusBadRequest)
	} else {
		// fully qualify the domain name if it is not already:
		if string(domain[len(domain)-1]) != "." {
			domain = fmt.Sprintf("%s.", domain)
		}

		nslog.Debug.Printf("adding domain entry: dns:%s:%s :: %s", dnsType, domain, domainValue)

		// add dns entry dns:<dns_type>:<domain> <domain_value>
		redisClient.Set(fmt.Sprintf("dns:%s:%s", dnsType, domain), domainValue, 0).Err()

		// return confirmation header to client
		w.Header().Set("x-register", "registered")
		w.WriteHeader(http.StatusOK)
	}
}

func dnsRmHandler(w http.ResponseWriter, r *http.Request, redisClient *redis.Client) {
	r.ParseForm()

	rmType := strings.ToLower(strings.TrimSpace(r.FormValue("dnstype")))
	rmDomain := strings.TrimSpace(r.FormValue("domain"))

	if len(rmDomain) == 0 {
		nslog.Debug.Println("domain not set, exiting..")
		w.WriteHeader(http.StatusBadRequest)
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
	} else {
		// just remove the specific type
		nslog.Debug.Printf("removing %s entry for %s", rmType, rmDomain)
		redisClient.Del(fmt.Sprintf("dns:%s:%s", rmType, rmDomain))
	}
	w.WriteHeader(http.StatusOK)
}

func apiHostsHandler(w http.ResponseWriter, r *http.Request, redisClient *redis.Client) {
	hosts, err := redisClient.SInter("index:live").Result()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
	}
	fmt.Fprintln(w, hosts)
}

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
	redisClient.Del("tmp:cluster:index")
	fmt.Fprintln(w, clusters)
}

func apiHostSpecHandler(w http.ResponseWriter, r *http.Request, redisClient *redis.Client) {
	cluster := strings.TrimSpace(r.FormValue("cluster"))
	host := strings.TrimSpace(r.FormValue("host"))

	ip, err := redisClient.Get(fmt.Sprintf("cluster:%s:%s", cluster, host)).Result()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
	}
	fmt.Fprintln(w, ip)
}
