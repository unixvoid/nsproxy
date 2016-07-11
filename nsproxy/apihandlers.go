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

	// make sure domain and value are set
	if (len(domain) == 0) || (len(domainValue) == 0) {
		nslog.Debug.Println("domain or value not set, exiting..")
		w.WriteHeader(http.StatusBadRequest)
	} else {
		if len(dnsType) == 0 {
			// default to aname entry
			dnsType = "a"
		} else {
			// if dnstype is set, make sure it is something we support
			switch dnsType {
			case
				"a",
				"aaaa",
				"cname":
				break
			default:
				nslog.Debug.Printf("unsupported dnstype '%s', exiting..\n", dnsType)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
		}
		if dnsType == "cname" {
			// if we are dealing with a CNAME entry fully qualify it
			if string(domainValue[len(domainValue)-1]) != "." {
				domainValue = fmt.Sprintf("%s.", domainValue)
			}
		}
		// fully qualify the domain name if it is not already:
		if string(domain[len(domain)-1]) != "." {
			domain = fmt.Sprintf("%s.", domain)
		}

		nslog.Debug.Printf("adding domain entry: dns:%s:%s :: %s", dnsType, domain, domainValue)

		// add dns entry dns:<dns_type>:<domain> <domain_value>
		redisClient.Set(fmt.Sprintf("dns:%s:%s", dnsType, domain), domainValue, 0).Err()

		// add dns entry to the list of custom dns names
		redisClient.SAdd("index:dns", fmt.Sprintf("%s:%s", dnsType, domain))

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

func apiHostsHandler(w http.ResponseWriter, r *http.Request, redisClient *redis.Client) {
	hosts, err := redisClient.SInter("index:live").Result()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
	}
	fmt.Fprintln(w, hosts)
}

func dnsHostsHandler(w http.ResponseWriter, r *http.Request, redisClient *redis.Client) {
	hosts, err := redisClient.SInter("index:dns").Result()
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
