package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"git.unixvoid.com/mfaltys/glogger"
	"git.unixvoid.com/mfaltys/nsproxy/nsmanager"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/miekg/dns"
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
		WebPollFreq       time.Duration
		WebHostPort       int
		WebHostDeadTime   time.Duration
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
	config   = Config{}
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

func main() {
	// init config file
	err := gcfg.ReadFileInto(&config, "data/config.gcfg")
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

	if config.Clustermanager.UseClusterManager {
		// start async cluster listener
		go asyncClusterListener()
	}

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
		// remove the FQDM '.' from end of 'hostname'
		fqdnHostname := hostname
		hostname := strings.Replace(hostname, ".", "", -1)

		// grab the first item in the list
		firstEntry, _ := redisClient.LIndex(fmt.Sprintf("list:%s", hostname), 0).Result()

		// return ip to client
		lookup, _ := nsmanager.ClusterQuery(hostname, firstEntry, redisClient)
		customRR := aBuilder(fqdnHostname, lookup)
		rep := new(dns.Msg)
		rep.SetReply(req)
		rep.Answer = append(rep.Answer, customRR)

		glogger.Debug.Println("serving", hostname, "from local record")
		w.WriteMsg(rep)

		// pop the list and add the entry to the end, it just got lb'd
		firstEntry, _ = redisClient.LPop(fmt.Sprintf("list:%s", hostname)).Result()
		redisClient.RPush(fmt.Sprintf("list:%s", hostname), firstEntry)
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
	rr := new(dns.A)
	rr.Hdr = dns.RR_Header{Name: hostname, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: config.Dns.Ttl}
	addr := strings.TrimSuffix(lookup, "\n")
	rr.A = net.ParseIP(addr)
	return rr
}

func aaaaBuilder(hostname, lookup string) *dns.AAAA {
	rr := new(dns.AAAA)
	rr.Hdr = dns.RR_Header{Name: hostname, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: config.Dns.Ttl}
	addr := strings.TrimSuffix(lookup, "\n")
	rr.AAAA = net.ParseIP(addr)
	return rr
}

func cnameBuilder(hostname, lookup string) *dns.CNAME {
	rr := new(dns.CNAME)
	rr.Hdr = dns.RR_Header{Name: hostname, Rrtype: dns.TypeCNAME, Class: dns.ClassINET, Ttl: config.Dns.Ttl}
	rr.Target = lookup
	return rr
}

func websocketHandler(redisClient *redis.Client) {
	indexFile, _ := os.Open("data/index.html")
	index, _ := ioutil.ReadAll(indexFile)
	styleFile, _ := os.Open("data/style.css")
	style, _ := ioutil.ReadAll(styleFile)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		var err error
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			glogger.Error.Println(err)
			return
		}
		liveHostString := ""
		tmpHostString := ""
		for {
			// need logic to only update socket on change
			time.Sleep(config.Clustermanager.WebPollFreq * time.Second)

			liveHosts, _ := redisClient.SInter("index:live").Result()
			tmpHostString = liveHostString
			liveHostString = ""
			for _, i := range liveHosts {
				// get the host ip
				ip, _ := redisClient.Get(fmt.Sprintf("cluster:%s", i)).Result()
				// drop the host followed by ip in this syntax: {host,ip host,ip}
				liveHostString = fmt.Sprintf("%s %s", liveHostString, fmt.Sprintf("%s,%s", i, ip))

			}
			// only update websocket if there is a change
			if liveHostString != tmpHostString {
				conn.WriteMessage(websocket.TextMessage, []byte(liveHostString))
			}
		}
	})
	http.HandleFunc("/ws2", func(w http.ResponseWriter, r *http.Request) {
		var err error
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			glogger.Error.Println(err)
			return
		}
		deadHostString := ""
		tmpHostString := ""
		for {
			// need logic to only update socket on change
			time.Sleep(config.Clustermanager.WebPollFreq * time.Second)

			deadHosts, _ := redisClient.SInter("index:dead").Result()
			tmpHostString = deadHostString
			deadHostString = ""
			for _, i := range deadHosts {
				// get the host ip
				ip, _ := redisClient.Get(fmt.Sprintf("cluster:%s", i)).Result()
				// drop the host followed by ip in this syntax: {host,ip host,ip}
				deadHostString = fmt.Sprintf("%s %s", deadHostString, fmt.Sprintf("%s,%s", i, ip))

			}
			// only update websocket if there is a change
			if deadHostString != tmpHostString {
				conn.WriteMessage(websocket.TextMessage, []byte(deadHostString))
			}
		}
	})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, string(index))
	})
	http.HandleFunc("/style.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css")
		fmt.Fprintf(w, string(style))
	})
	port := fmt.Sprint(":", config.Clustermanager.WebHostPort)
	http.ListenAndServe(port, nil)
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

	// start async websocket server
	go websocketHandler(redisClient)

	// check index:master against index:live
	clusterDiff(redisClient)
	log.Fatal(http.ListenAndServe(port, router))
}

func clusterHandler(w http.ResponseWriter, r *http.Request, redisClient *redis.Client) {
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
		syncList(cluster, redisClient)
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
	//redisClient.Del(fmt.Sprintf("cluster:%s:%s", cluster, hostname))
	redisClient.Expire(fmt.Sprintf("cluster:%s:%s", cluster, hostname), (config.Clustermanager.WebHostDeadTime * time.Minute)).Err()
	// add the host to the removed cluster entry
	redisClient.SAdd("index:dead", fmt.Sprintf("%s:%s", cluster, hostname))
	syncDead(redisClient)

	// remove the index entry, it is no longer in the cluster
	redisClient.SRem(fmt.Sprintf("index:cluster:%s", cluster), hostname)
	syncList(cluster, redisClient)

	// remove the host form master and live index entries
	redisClient.SRem("index:master", fmt.Sprintf("%s:%s", cluster, hostname))
	redisClient.SRem("index:live", fmt.Sprintf("%s:%s", cluster, hostname))
}

func clusterDiff(redisClient *redis.Client) {
	// 'sdiff index:master index:live' will return the set of hosts
	// that do not have listeners currently attached
	glogger.Debug.Println("diffing cluster")
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

func syncList(cluster string, redisClient *redis.Client) {
	// sync a cluster index entry to a list. the redis set is used for speed, and the
	// redis list (sorted set) is used for load balancer algorithms
	glogger.Debug.Println("syncing list")
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

func syncDead(redisClient *redis.Client) {
	// for every line in index:dead, we need to remove any entries that no longer
	// exist. these keys are set to expire every 20 mins, so we clean up the index.
	indexString, _ := redisClient.SInter("index:dead").Result()
	for _, i := range indexString {
		_, err := redisClient.Get(fmt.Sprintf("cluster:%s", i)).Result()
		if err != nil {
			// remove the entry from index:dead
			redisClient.LRem("index:dead", -4, i)
		}
	}
}
