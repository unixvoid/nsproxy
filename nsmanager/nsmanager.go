package nsmanager

import (
	"errors"
	"fmt"
	"github.com/tatsushid/go-fastping"
	"gopkg.in/redis.v3"
	"net"
	"os"
	"time"
)

func Query(queryType, recordType, queryAttribute string, redisClient *redis.Client) (string, error) {
	searchString := fmt.Sprintf("%s:%s:%s", queryType, recordType, queryAttribute)
	val, err := redisClient.Get(searchString).Result()
	if err != nil {
		return "", errors.New("string not found.")
	} else {
		return val, nil
	}
	return "", nil
}

func PingHost(hostIp string) bool {
	var online bool
	p := fastping.NewPinger()
	ra, err := net.ResolveIPAddr("ip4:icmp", hostIp)
	//ra, err := net.ResolveIPAddr("udp4", os.Args[1])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	p.AddIPAddr(ra)
	p.OnRecv = func(addr *net.IPAddr, rtt time.Duration) {
		//fmt.Printf("IP Addr: %s receive, RTT: %v\n", addr.String(), rtt)
		online = true
	}
	err = p.Run()
	if err != nil {
		fmt.Println(err)
	}
	if online {
		return true
	} else {
		return false
	}
}
