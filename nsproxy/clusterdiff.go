package main

import (
	"fmt"
	"strings"

	"github.com/unixvoid/nsproxy/pkg/nslog"
	"gopkg.in/redis.v3"
)

func clusterDiff(redisClient *redis.Client) {
	// 'sdiff index:master index:live' will return the set of hosts
	// that do not have listeners currently attached
	nslog.Debug.Println("diffing cluster")
	diffString := redisClient.SDiff("index:master", "index:live")
	tmp, _ := diffString.Result()
	// for ever entry that is not in index:live
	for _, b := range tmp {
		nslog.Debug.Println("found diff for:", b)
		s := strings.SplitN(b, ":", 2)
		cluster, hostname := s[0], s[1]
		ip, _ := redisClient.Get(fmt.Sprintf("cluster:%s:%s", cluster, hostname)).Result()
		port, _ := redisClient.Get(fmt.Sprintf("port:%s:%s", cluster, hostname)).Result()
		// spawn cluster manager for host
		go spawnClusterManager(cluster, hostname, ip, port, redisClient)
		// add host to live entry now
		redisClient.SAdd("index:live", fmt.Sprintf("%s:%s", cluster, hostname))
	}
}
