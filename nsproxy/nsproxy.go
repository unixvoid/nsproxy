package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/miekg/dns"
	"github.com/unixvoid/nsproxy/pkg/nslog"
	"github.com/unixvoid/nsproxy/pkg/nsmanager"
	"gopkg.in/gcfg.v1"
	"gopkg.in/redis.v3"
)

type Config struct {
	Server struct {
		Port     int
		Loglevel string
	}
	Clustermanager struct {
		UseClusterManager bool
		Port              int
		PingFreq          time.Duration
		ClientPingType    string
		ConnectionDrain   int
	}
	Dns struct {
		Ttl uint32
	}
	Upstreamdns struct {
		Server string
	}
	Redis struct {
		Host     string
		Password string
	}
}

var (
	config = Config{}
)

func main() {
	// read in conf
	readConf()

	// init logger
	initLogger()

	// init redis connection
	redisClient, redisErr := initRedisConnection()
	if redisErr != nil {
		nslog.Error.Println("redis connection cannot be made.")
		nslog.Error.Println("nsproxy will continue to function in passthrough mode only")
	} else {
		nslog.Debug.Println("connection to redis succeeded.")
		if config.Clustermanager.UseClusterManager {
			// start async cluster listener
			go asyncClusterListener()
		}
	}

	// format the string to be :port
	port := fmt.Sprint(":", config.Server.Port)

	udpServer := &dns.Server{Addr: port, Net: "udp"}
	tcpServer := &dns.Server{Addr: port, Net: "tcp"}
	nslog.Info.Println("started server on", config.Server.Port)
	dns.HandleFunc(".", func(w dns.ResponseWriter, req *dns.Msg) {
		route(w, req, redisClient)
	})

	go func() {
		log.Fatal(udpServer.ListenAndServe())
	}()
	log.Fatal(tcpServer.ListenAndServe())
}
func readConf() {
	// init config file
	err := gcfg.ReadFileInto(&config, "config.gcfg")
	if err != nil {
		fmt.Printf("Could not load config.gcfg, error: %s\n", err)
		return
	}
}

func initLogger() {
	// init logger
	if config.Server.Loglevel == "debug" {
		nslog.LogInit(os.Stdout, os.Stdout, os.Stdout, os.Stderr)
	} else if config.Server.Loglevel == "cluster" {
		nslog.LogInit(os.Stdout, os.Stdout, ioutil.Discard, os.Stderr)
	} else if config.Server.Loglevel == "info" {
		nslog.LogInit(os.Stdout, ioutil.Discard, ioutil.Discard, os.Stderr)
	} else {
		nslog.LogInit(ioutil.Discard, ioutil.Discard, ioutil.Discard, os.Stderr)
	}
}

func initRedisConnection() (*redis.Client, error) {
	// init redis connection
	redisClient := redis.NewClient(&redis.Options{
		Addr:     config.Redis.Host,
		Password: config.Redis.Password,
		DB:       0,
	})

	_, redisErr := redisClient.Ping().Result()
	return redisClient, redisErr
}

func route(w dns.ResponseWriter, req *dns.Msg, redisClient *redis.Client) {
	// async run proxy task
	go proxy(config.Upstreamdns.Server, w, req, redisClient)
}

func proxy(addr string, w dns.ResponseWriter, req *dns.Msg, redisClient *redis.Client) {
	clusterString := req.Question[0].Name
	if strings.Contains(clusterString, "cluster-") {
		// it is a cluster entry, forward the request to the dns cluster handler
		// remove the FQDM '.' from end of 'clusterString'
		fqdnHostname := clusterString
		// redo syntax to be cluster:<cluster>
		clusterString = strings.Replace(clusterString, "-", ":", 1)
		clusterString = strings.Replace(clusterString, ".", "", -1)

		// parse out 'cluster:'
		s := strings.SplitN(clusterString, ":", 2)
		clusterName := s[1]

		// grab the first item in the list
		hostName, _ := redisClient.LIndex(fmt.Sprintf("list:%s", clusterString), 0).Result()
		hostIp, _ := nsmanager.ClusterQuery(clusterString, hostName, redisClient)
		hostCWeight, _ := redisClient.Get(fmt.Sprintf("cweight:%s:%s", clusterName, hostName)).Result()
		hostCWeightNum, _ := strconv.Atoi(hostCWeight)

		// return ip to client
		lookup := hostIp
		customRR := aBuilder(fqdnHostname, lookup)
		rep := new(dns.Msg)
		rep.SetReply(req)
		rep.Answer = append(rep.Answer, customRR)

		// if we dont have an entry, pop a NXDOMAIN error
		if len(hostIp) == 0 {
			rep.Rcode = dns.RcodeNameError
		}

		hostCWeightNum = hostCWeightNum - 1

		if hostCWeightNum <= 0 {
			// pop the list and add the entry to the end, it just got lb'd
			hostIp, _ = redisClient.LPop(fmt.Sprintf("list:%s", clusterString)).Result()
			redisClient.RPush(fmt.Sprintf("list:%s", clusterString), hostIp)
			hostWeight, _ := redisClient.Get(fmt.Sprintf("weight:%s:%s", clusterName, hostName)).Result()
			hostWeightNum, _ := strconv.Atoi(hostWeight)
			hostCWeightNum = hostWeightNum
			nslog.Debug.Println("resetting host weight")
		}
		nslog.Debug.Println("serving", clusterString, "from local record")
		w.WriteMsg(rep)

		redisClient.Set(fmt.Sprintf("cweight:%s:%s", clusterName, hostName), hostCWeightNum, 0).Err()
	} else {

		transport := "udp"
		if _, ok := w.RemoteAddr().(*net.TCPAddr); ok {
			transport = "tcp"
		}
		c := &dns.Client{Net: transport}
		resp, _, err := c.Exchange(req, addr)

		if err != nil {
			nslog.Debug.Println(err)
			dns.HandleFailed(w, req)
			return
		}

		// call main builder to craft and send the response
		mainBuilder(w, req, resp, clusterString, redisClient)
	}
}
