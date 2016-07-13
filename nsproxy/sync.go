package main

import (
	"fmt"

	"github.com/unixvoid/nsproxy/pkg/nslog"
	"gopkg.in/redis.v3"
)

func syncList(cluster string, redisClient *redis.Client) {
	// sync a cluster index entry to a list. the redis set is used for speed, and the
	// redis list (sorted set) is used for load balancer algorithms
	nslog.Debug.Println("syncing list")
	indexString, _ := redisClient.SInter(fmt.Sprintf("index:cluster:%s", cluster)).Result()
	// populate a tmp list
	for _, i := range indexString {
		redisClient.RPush(fmt.Sprintf("tmp:list:cluster:%s", cluster), i)
	}
	// delete current list
	redisClient.Del(fmt.Sprintf("list:cluster:%s", cluster))
	// move tmp list to current
	redisClient.Rename(fmt.Sprintf("tmp:list:cluster:%s", cluster), fmt.Sprintf("list:cluster:%s", cluster))
}
