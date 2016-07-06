package main

import (
	"fmt"
	"time"

	"github.com/unixvoid/nsproxy/pkg/nslog"
	"github.com/unixvoid/nsproxy/pkg/nsmanager"
	"gopkg.in/redis.v3"
)

func spawnClusterManager(cluster, hostname, ip, port string, redisClient *redis.Client) {
	// add in a connection drain redis entry cluster:<cluster_name>:<hostname> <drain time>
	connectionDrain := config.Clustermanager.ConnectionDrain
	if config.Clustermanager.ClientPingType == "port" {
		nslog.Cluster.Printf("spawning async cluster manager for %s:%s on port %s", cluster, hostname, port)
	} else {
		nslog.Cluster.Printf("spawning async cluster manager for %s:%s", cluster, hostname)
	}

	var healthCheck bool
	online := true
	for online {
		if config.Clustermanager.ClientPingType == "port" {
			healthCheck, _ = nsmanager.HealthCheck(ip, port)
		} else {
			healthCheck = nsmanager.PingHost(ip)
		}
		if healthCheck {
			//nslog.Debug.Printf("- %s:%s online", cluster, hostname)
			// reset connection drain
			if connectionDrain != config.Clustermanager.ConnectionDrain {
				nslog.Cluster.Printf("%s:%s listener draining reset", cluster, hostname)
				connectionDrain = config.Clustermanager.ConnectionDrain
			}
		} else {
			if connectionDrain < (0 + int(config.Clustermanager.PingFreq)) {
				nslog.Debug.Printf("- %s:%s offline", cluster, hostname)
				online = false
				break
			}
			// print draining message if first shot
			if connectionDrain == config.Clustermanager.ConnectionDrain {
				nslog.Cluster.Printf("%s:%s listener draining", cluster, hostname)
			}
			connectionDrain = connectionDrain - int(config.Clustermanager.PingFreq)
		}
		// time between host pings
		time.Sleep(time.Second * config.Clustermanager.PingFreq)
	}

	nslog.Cluster.Printf("closing %s:%s listener", cluster, hostname)
	// remove the server entry, it is no longer online
	redisClient.Del(fmt.Sprintf("cluster:%s:%s", cluster, hostname))

	// remove the weight entries, it is no longer online
	redisClient.Del(fmt.Sprintf("weight:%s:%s", cluster, hostname))
	redisClient.Del(fmt.Sprintf("cweight:%s:%s", cluster, hostname))

	// remove the port entry, it is no longer online
	redisClient.Del(fmt.Sprintf("port:%s:%s", cluster, hostname))

	// remove the index entry, it is no longer in the cluster
	redisClient.SRem(fmt.Sprintf("index:cluster:%s", cluster), hostname)
	syncList(cluster, redisClient)

	// remove the host form master and live index entries
	redisClient.SRem("index:master", fmt.Sprintf("%s:%s", cluster, hostname))
	redisClient.SRem("index:live", fmt.Sprintf("%s:%s", cluster, hostname))
}
