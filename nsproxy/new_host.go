package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/unixvoid/nsproxy/pkg/nslog"
	"gopkg.in/redis.v3"
)

func clusterHandler(w http.ResponseWriter, r *http.Request, redisClient *redis.Client) {
	ip := strings.Split(r.RemoteAddr, ":")[0]
	var hostWeight string

	r.ParseForm()
	hostname := strings.TrimSpace(r.FormValue("hostname"))
	cluster := strings.TrimSpace(r.FormValue("cluster"))
	hostIp := strings.TrimSpace(r.FormValue("ip"))
	hostPort := strings.TrimSpace(r.FormValue("port"))
	if len(strings.TrimSpace(r.FormValue("weight"))) == 0 {
		hostWeight = "1"
	} else {
		hostWeight = strings.TrimSpace(r.FormValue("weight"))
	}

	// use parsed ip if it is set
	if len(hostIp) != 0 {
		ip = hostIp
	}

	// make sure hostname and cluster are set
	if (len(hostname) == 0) || (len(cluster) == 0) {
		nslog.Debug.Println("hostame or cluster not set, exiting..")
		w.WriteHeader(http.StatusBadRequest)
	} else {
		nslog.Debug.Printf("registing %s:%s :: %s", cluster, hostname, ip)

		// add cluster entry cluster:<cluster_name>:<hostname> <ip>
		redisClient.Set(fmt.Sprintf("cluster:%s:%s", cluster, hostname), ip, 0).Err()

		// add weight to client
		redisClient.Set(fmt.Sprintf("weight:%s:%s", cluster, hostname), hostWeight, 0).Err()
		redisClient.Set(fmt.Sprintf("cweight:%s:%s", cluster, hostname), hostWeight, 0).Err()

		// add to index if it does not exist index:cluster:<cluster_name> <host_name>
		redisClient.SAdd(fmt.Sprintf("index:cluster:%s", cluster), hostname)
		go syncList(cluster, redisClient)
		redisClient.SAdd("index:master", fmt.Sprintf("%s:%s", cluster, hostname))

		// add port if it is set
		if len(hostPort) != 0 && config.Clustermanager.ClientPingType == "port" {
			redisClient.Set(fmt.Sprintf("port:%s:%s", cluster, hostname), hostPort, 0).Err()
		}

		// diff index:master and index:live to find/register the new live host
		go clusterDiff(redisClient)

		// remove any state entry that may exist
		redisClient.SRem(fmt.Sprintf("state:cluster:%s", cluster), fmt.Sprintf("%s:%s", hostIp, hostPort))

		// return confirmation header to client
		w.Header().Set("x-register", "registered")
		w.WriteHeader(http.StatusOK)
	}
}
