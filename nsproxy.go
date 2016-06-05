package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"gopkg.in/gcfg.v1"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"git.unixvoid.com/mfaltys/glogger"
	"git.unixvoid.com/mfaltys/nsproxy/nsmanager"
	"github.com/miekg/dns"
	"gopkg.in/redis.v3"
)

type Config struct {
	Server struct {
		Port     int
		Loglevel string
	}
	Clustermanager struct {
		Port       int
		HostTTL    time.Duration
		ClusterTTL time.Duration
		PingFreq   time.Duration
	}
	Dns struct {
		Ttl uint32
	}
	Upstreamdns struct {
		Server string
		Port   int
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
	// init config file
	err := gcfg.ReadFileInto(&config, "config.gcfg")
	if err != nil {
		fmt.Printf("Could not load config.gcfg, error: %s\n", err)
		return
	}

	// init logger
	if config.Server.Loglevel == "debug" {
		glogger.LogInit(os.Stdout, os.Stdout, os.Stdout, os.Stderr)
	} else if config.Server.Loglevel == "cluster" {
		glogger.LogInit(os.Stdout, os.Stdout, ioutil.Discard, os.Stderr)
	} else if config.Server.Loglevel == "info" {
		glogger.LogInit(os.Stdout, ioutil.Discard, ioutil.Discard, os.Stderr)
	} else {
		glogger.LogInit(ioutil.Discard, ioutil.Discard, ioutil.Discard, os.Stderr)
	}

	// start async cluster listener
	go asyncClusterListener()

	// format the string to be :port
	port := fmt.Sprint(":", config.Server.Port)

	udpServer := &dns.Server{Addr: port, Net: "udp"}
	tcpServer := &dns.Server{Addr: port, Net: "tcp"}
	glogger.Info.Println("started server on", config.Server.Port)
	dns.HandleFunc(".", route)

	go func() {
		log.Fatal(udpServer.ListenAndServe())
	}()
	log.Fatal(tcpServer.ListenAndServe())
}

func route(w dns.ResponseWriter, req *dns.Msg) {
	// init redis connection
	redisClient := redis.NewClient(&redis.Options{
		Addr:     config.Redis.Host,
		Password: config.Redis.Password,
		DB:       0,
	})

	proxy(config.Upstreamdns.Server, w, req, redisClient)
}

func proxy(addr string, w dns.ResponseWriter, req *dns.Msg, redisClient *redis.Client) {
	hostname := req.Question[0].Name
	if strings.Contains(hostname, "cluster:") {
		// it is a cluster entry, forward the request to the dns cluster handler
		// cluster:coreos
		//indexString := redisClient.SInter(fmt.Sprintf("index:%s", hostname))
		// CURRENT i just made syncLists()
		indexString, _ := redisClient.SInter("index:cluster:coreos").Result()
		glogger.Cluster.Println(indexString[(len(indexString) - 1)])
		// remove and add back to the end of set
		redisClient.SRem("index:cluster:coreos", indexString[(len(indexString)-1)])
		redisClient.SAdd("index:cluster:coreos", indexString[(len(indexString)-1)])

		indexString, _ = redisClient.SInter("index:cluster:coreos").Result()
		glogger.Cluster.Println(indexString[len(indexString)-1])

		//redisClient.SRem("index:live", fmt.Sprintf("%s:%s", cluster, hostname))
		//redisClient.SAdd("index:live", fmt.Sprintf("%s:%s", cluster, hostname))
		//ip, _ := redisClient.Get(fmt.Sprintf("cluster:%s:%s", cluster, hostname)).Result()
	} else {

		transport := "udp"
		if _, ok := w.RemoteAddr().(*net.TCPAddr); ok {
			transport = "tcp"
		}
		c := &dns.Client{Net: transport}
		resp, _, err := c.Exchange(req, addr)

		if err != nil {
			glogger.Error.Println(err)
			dns.HandleFailed(w, req)
			return
		}

		// call main builder to craft and send the response
		mainBuilder(w, req, resp, hostname, redisClient)
	}
}

func mainBuilder(w dns.ResponseWriter, req, resp *dns.Msg, hostname string, redisClient *redis.Client) {
	var (
		customRR  dns.RR
		lookupErr error
		lookup    string
	)

	// supporting only one query per request right now. ie only req.Question[0]
	switch req.Question[0].Qtype {
	case 1:
		lookup, lookupErr = nsmanager.Query("dns", "a", hostname, redisClient)
		customRR = aBuilder(hostname, lookup)
		break
	case 5:
		lookup, lookupErr = nsmanager.Query("dns", "cname", hostname, redisClient)
		customRR = cnameBuilder(hostname, lookup)
		break
	case 28:
		lookup, lookupErr = nsmanager.Query("dns", "aaaa", hostname, redisClient)
		customRR = aaaaBuilder(hostname, lookup)
		break
	default:
		lookup, lookupErr = nsmanager.Query("dns", "a", hostname, redisClient)
		customRR = aBuilder(hostname, lookup)
		break
	}

	if lookupErr == nil {
		// create the response and append the crafted a portion to it
		rep := new(dns.Msg)
		rep.SetReply(req)
		rep.Answer = append(rep.Answer, customRR)

		glogger.Debug.Println("serving", hostname, "from local record")
		w.WriteMsg(rep)
		return
	}
	glogger.Debug.Println("serving", hostname, "from", config.Upstreamdns.Server)
	w.WriteMsg(resp)
}

func aBuilder(hostname, lookup string) *dns.A {
	// craft the A record response
	rr := new(dns.A)
	rr.Hdr = dns.RR_Header{Name: hostname, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: config.Dns.Ttl}
	addr := strings.TrimSuffix(lookup, "\n")
	rr.A = net.ParseIP(addr)
	return rr
}

func aaaaBuilder(hostname, lookup string) *dns.AAAA {
	// craft the A record response
	rr := new(dns.AAAA)
	rr.Hdr = dns.RR_Header{Name: hostname, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: config.Dns.Ttl}
	addr := strings.TrimSuffix(lookup, "\n")
	rr.AAAA = net.ParseIP(addr)
	return rr
}

func cnameBuilder(hostname, lookup string) *dns.CNAME {
	// craft the A record response
	rr := new(dns.CNAME)
	rr.Hdr = dns.RR_Header{Name: hostname, Rrtype: dns.TypeCNAME, Class: dns.ClassINET, Ttl: config.Dns.Ttl}
	rr.Target = lookup
	return rr
}

func asyncClusterListener() {
	// async listener gets its own redis connection
	redisClient := redis.NewClient(&redis.Options{
		Addr:     config.Redis.Host,
		Password: config.Redis.Password,
		DB:       0,
	})
	// first boot, remove live file
	redisClient.Del(fmt.Sprintf("index:live"))

	// format the string to be :port
	port := fmt.Sprint(":", config.Clustermanager.Port)

	glogger.Info.Println("started async cluster listener on port", config.Clustermanager.Port)
	router := mux.NewRouter()
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		clusterHandler(w, r, redisClient)
	})

	clusterDiff(redisClient)
	log.Fatal(http.ListenAndServe(port, router))
}

func clusterHandler(w http.ResponseWriter, r *http.Request, redisClient *redis.Client) {
	// curl -d hostname=testname -d cluster=testcluster http://localhost:8080
	ip := strings.Split(r.RemoteAddr, ":")[0]

	r.ParseForm()
	hostname := strings.TrimSpace(r.FormValue("hostname"))
	cluster := strings.TrimSpace(r.FormValue("cluster"))
	// make sure hostname and cluster are set
	if (len(hostname) == 0) || (len(cluster) == 0) {
		glogger.Debug.Println("hostame or cluster not set, exiting..")
	} else {
		glogger.Debug.Printf("registing %s:%s :: %s", cluster, hostname, ip)

		// add cluster entry cluster:<cluster_name>:<hostname> <ip>
		redisClient.Set(fmt.Sprintf("cluster:%s:%s", cluster, hostname), ip, 0).Err()

		// add to index if it does not exist index:cluster:<cluster_name> <host_name>
		redisClient.SAdd(fmt.Sprintf("index:cluster:%s", cluster), hostname)
		redisClient.SAdd("index:master", fmt.Sprintf("%s:%s", cluster, hostname))

		// diff index:master and index:live to find/register the new live host
		clusterDiff(redisClient)

		// return confirmation header to client
		w.Header().Set("x-register", "registered")
	}
}

func spawnClusterManager(cluster, hostname, ip string, redisClient *redis.Client) {
	glogger.Cluster.Printf("spawning async cluster manager for %s:%s", cluster, hostname)
	online := true
	for online {
		if nsmanager.PingHost(ip) {
			glogger.Debug.Printf("- %s:%s online", cluster, hostname)
		} else {
			glogger.Debug.Printf("- %s:%s offline", cluster, hostname)
			online = false
			break
		}
		// time between host pings
		time.Sleep(time.Second * config.Clustermanager.PingFreq)
	}
	glogger.Cluster.Printf("closing %s:%s listener", cluster, hostname)
	// remove the server entry, it is no longer online
	redisClient.Del(fmt.Sprintf("cluster:%s:%s", cluster, hostname))

	// remove the index entry, it is no longer in the cluster
	redisClient.SRem(fmt.Sprintf("index:cluster:%s", cluster), hostname)

	redisClient.SRem("index:master", fmt.Sprintf("%s:%s", cluster, hostname))
	redisClient.SRem("index:live", fmt.Sprintf("%s:%s", cluster, hostname))
}

func clusterDiff(redisClient *redis.Client) {
	// 'sdiff index:master index:live' will return the set of hosts
	// that do not have listeners currently attached
	glogger.Cluster.Println("diffing cluster")
	diffString := redisClient.SDiff("index:master", "index:live")
	tmp, _ := diffString.Result()
	// for ever entry that is not in index:live
	for _, b := range tmp {
		glogger.Debug.Println("found diff for:", b)
		s := strings.SplitN(b, ":", 2)
		cluster, hostname := s[0], s[1]
		ip, _ := redisClient.Get(fmt.Sprintf("cluster:%s:%s", cluster, hostname)).Result()
		// spawn cluster manager for host
		go spawnClusterManager(cluster, hostname, ip, redisClient)
		// add host to live entry now
		redisClient.SAdd("index:live", fmt.Sprintf("%s:%s", cluster, hostname))
	}
}

// TODO add sync function to sync 'index:cluster:<cluster_name>' and 'list:cluster:<cluster_name>'
// we use the list to load balance (aka order matters)
// this will get synced every time a element is added or subtracted from the index
func syncList(cluster string, redisClient *redis.Client) {
	glogger.Cluster.Println("syncing list")
	indexString, _ := redisClient.SInter(fmt.Sprintf("index:cluster:%s", cluster)).Result()
	for _, i := range indexString {
		redisClient.RPush(fmt.Sprintf("tmp:list:cluster:%s", cluster), i)
	}
	redisClient.Del(fmt.Sprintf("list:cluster:%s", cluster))
	redisClient.Rename(fmt.Sprintf("tmp:list:cluster:%s", cluster), fmt.Sprintf("list:cluster:%s", cluster))
}
