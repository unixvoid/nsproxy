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
		Port int
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
		glogger.LogInit(os.Stdout, os.Stdout, os.Stderr)
	} else if config.Server.Loglevel == "info" {
		glogger.LogInit(os.Stdout, ioutil.Discard, os.Stderr)
	} else {
		glogger.LogInit(ioutil.Discard, ioutil.Discard, os.Stderr)
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

	// format the string to be :port
	port := fmt.Sprint(":", config.Clustermanager.Port)

	glogger.Info.Println("started async cluster listener on port", config.Clustermanager.Port)
	router := mux.NewRouter()
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		clusterHandler(w, r, redisClient)
	})

	log.Fatal(http.ListenAndServe(port, router))
}

func clusterHandler(w http.ResponseWriter, r *http.Request, redisClient *redis.Client) {
	// curl -d hostname=testName http://localhost:8080
	ip := strings.Split(r.RemoteAddr, ":")[0]

	r.ParseForm()
	hostname := strings.TrimSpace(r.FormValue("hostname"))
	cluster := strings.TrimSpace(r.FormValue("cluster"))
	// make sure hostname and cluster are set
	if len(hostname) == 0 {
		glogger.Debug.Println("hostname not set, using 'defaulthost'")
		hostname = "defaulthost"
	}
	if len(cluster) == 0 {
		glogger.Debug.Println("cluster not set, using 'defaultcluster'")
		cluster = "defaultcluster"
	}
	glogger.Debug.Printf("registered %s:%s :: %s", cluster, hostname, ip)

	// all add to its own tag.. cluster:<cluster_name>:<hostname>
	clusterStr := fmt.Sprintf("%s:%s:%s", "cluster", cluster, hostname)
	redisClient.Set(clusterStr, ip, 0).Err()

	// if it doesn't already exist in the index, add it
	indexStr := fmt.Sprintf("%s:%s:%s", "index", "cluster", cluster)
	searchHostname := fmt.Sprintf(" %s ", hostname)
	paddedHostname := fmt.Sprintf("%s ", hostname)
	index, err := redisClient.Get(indexStr).Result()
	if err != nil {
		// index does not exist, create it..
		redisClient.Set(indexStr, " ", 0).Err()
	}
	if strings.Contains(index, searchHostname) {
		// we now append the server hostname to cluster index.. index:cluster:<cluster_name>
		glogger.Debug.Println("host already exists in cluster index")
	} else {
		redisClient.Append(indexStr, paddedHostname)
	}

	// return msg to client
	w.Header().Set("x-register", "registered")
}
