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
