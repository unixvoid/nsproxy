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

	// init redis connection
	redisClient := redis.NewClient(&redis.Options{
		Addr:     config.Redis.Host,
		Password: config.Redis.Password,
		DB:       0,
	})

	_, redisErr := redisClient.Ping().Result()
	if redisErr != nil {
		glogger.Error.Println("redis connection cannot be made.")
		glogger.Error.Println("nsproxy will continue to function in passthrough mode only")
	} else {
		glogger.Debug.Println("connection to redis succeeded.")
		if config.Clustermanager.UseClusterManager {
			// start async cluster listener
			go asyncClusterListener()
		}
	}

	// format the string to be :port
	port := fmt.Sprint(":", config.Server.Port)

	udpServer := &dns.Server{Addr: port, Net: "udp"}
	tcpServer := &dns.Server{Addr: port, Net: "tcp"}
	glogger.Info.Println("started server on", config.Server.Port)
	dns.HandleFunc(".", func(w dns.ResponseWriter, req *dns.Msg) {
		route(w, req, redisClient)
	})

	go func() {
		log.Fatal(udpServer.ListenAndServe())
	}()
	log.Fatal(tcpServer.ListenAndServe())
}

func route(w dns.ResponseWriter, req *dns.Msg, redisClient *redis.Client) {
	proxy(config.Upstreamdns.Server, w, req, redisClient)
}

func proxy(addr string, w dns.ResponseWriter, req *dns.Msg, redisClient *redis.Client) {
	hostname := req.Question[0].Name
	if strings.Contains(hostname, "cluster-") {
		// it is a cluster entry, forward the request to the dns cluster handler
		// remove the FQDM '.' from end of 'hostname'
		fqdnHostname := hostname
		// redo syntax to be cluster:
		hostname := strings.Replace(hostname, "-", ":", 1)
		hostname = strings.Replace(hostname, ".", "", -1)

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
			glogger.Debug.Println(err)
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
	}).Methods("POST")
	router.HandleFunc("/dns", func(w http.ResponseWriter, r *http.Request) {
		dnsHandler(w, r, redisClient)
	}).Methods("POST")
	router.HandleFunc("/dns/rm", func(w http.ResponseWriter, r *http.Request) {
		dnsRmHandler(w, r, redisClient)
	}).Methods("POST")
	router.HandleFunc("/clusterspec", func(w http.ResponseWriter, r *http.Request) {
		apiClusterSpecHandler(w, r, redisClient)
	}).Methods("POST")
	router.HandleFunc("/hostspec", func(w http.ResponseWriter, r *http.Request) {
		apiHostSpecHandler(w, r, redisClient)
	}).Methods("POST")
	router.HandleFunc("/hosts", func(w http.ResponseWriter, r *http.Request) {
		apiHostsHandler(w, r, redisClient)
	}).Methods("GET")
	router.HandleFunc("/clusters", func(w http.ResponseWriter, r *http.Request) {
		apiClustersHandler(w, r, redisClient)
	}).Methods("GET")

	// check index:master against index:live
	clusterDiff(redisClient)
	log.Fatal(http.ListenAndServe(port, router))
}

func clusterHandler(w http.ResponseWriter, r *http.Request, redisClient *redis.Client) {
	ip := strings.Split(r.RemoteAddr, ":")[0]

	r.ParseForm()
	hostname := strings.TrimSpace(r.FormValue("hostname"))
	cluster := strings.TrimSpace(r.FormValue("cluster"))
	hostIp := strings.TrimSpace(r.FormValue("ip"))
	hostPort := strings.TrimSpace(r.FormValue("port"))

	// use parsed ip if it is set
	if len(hostIp) != 0 {
		ip = hostIp
	}

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

		// add port if it is set
		if len(hostPort) != 0 && config.Clustermanager.ClientPingType == "port" {
			redisClient.Set(fmt.Sprintf("port:%s:%s", cluster, hostname), hostPort, 0).Err()
		}

		// diff index:master and index:live to find/register the new live host
		clusterDiff(redisClient)

		// return confirmation header to client
		w.Header().Set("x-register", "registered")
	}
}

func dnsHandler(w http.ResponseWriter, r *http.Request, redisClient *redis.Client) {

	r.ParseForm()

	dnsType := strings.ToLower(strings.TrimSpace(r.FormValue("dnstype")))
	domain := strings.TrimSpace(r.FormValue("domain"))
	domainValue := strings.TrimSpace(r.FormValue("value"))
	if len(dnsType) == 0 {
		// default to aname entry
		dnsType = "a"
	}

	if dnsType == "cname" {
		// if we are dealing with a CNAME entry fully qualify it
		if string(domainValue[len(domainValue)-1]) != "." {
			domainValue = fmt.Sprintf("%s.", domainValue)
		}
	}

	// make sure domain and value are set
	if (len(domain) == 0) || (len(domainValue) == 0) {
		glogger.Debug.Println("domain or value not set, exiting..")
	} else {
		// fully qualify the domain name if it is not already:
		if string(domain[len(domain)-1]) != "." {
			domain = fmt.Sprintf("%s.", domain)
		}

		glogger.Debug.Printf("adding domain entry: dns:%s:%s :: %s", dnsType, domain, domainValue)

		// add dns entry dns:<dns_type>:<domain> <domain_value>
		redisClient.Set(fmt.Sprintf("dns:%s:%s", dnsType, domain), domainValue, 0).Err()

		// return confirmation header to client
		w.Header().Set("x-register", "registered")
	}
}

func dnsRmHandler(w http.ResponseWriter, r *http.Request, redisClient *redis.Client) {
	r.ParseForm()

	rmType := strings.ToLower(strings.TrimSpace(r.FormValue("dnstype")))
	rmDomain := strings.TrimSpace(r.FormValue("domain"))

	// fully qualify domain if not done already
	if string(rmDomain[len(rmDomain)-1]) != "." {
		rmDomain = fmt.Sprintf("%s.", rmDomain)
	}

	if len(rmType) == 0 {
		// if type not set, nix them all
		glogger.Debug.Printf("removing all dns types for %s", rmDomain)
		redisClient.Del(fmt.Sprintf("dns:a:%s", rmDomain))
		redisClient.Del(fmt.Sprintf("dns:aaaa:%s", rmDomain))
		redisClient.Del(fmt.Sprintf("dns:cname:%s", rmDomain))
	} else {
		// just remove the specific type
		glogger.Debug.Printf("removing %s entry for %s", rmType, rmDomain)
		redisClient.Del(fmt.Sprintf("dns:%s:%s", rmType, rmDomain))
	}
}

func apiHostsHandler(w http.ResponseWriter, r *http.Request, redisClient *redis.Client) {
	hosts, _ := redisClient.SInter("index:live").Result()
	fmt.Fprintln(w, hosts)
}

func apiClustersHandler(w http.ResponseWriter, r *http.Request, redisClient *redis.Client) {
	hosts, _ := redisClient.SInter("index:live").Result()
	for _, i := range hosts {
		// we now break at ':' and save the clusters piece
		s := strings.SplitN(i, ":", 2)
		// toss them all into a tmp redis set
		redisClient.SAdd("tmp:cluster:index", s[0])
	}
	// grab the set and delete
	clusters, _ := redisClient.SInter("tmp:cluster:index").Result()
	redisClient.Del("tmp:cluster:index")
	fmt.Fprintln(w, clusters)
}

func apiClusterSpecHandler(w http.ResponseWriter, r *http.Request, redisClient *redis.Client) {
	queryCluster := strings.TrimSpace(r.FormValue("cluster"))
	hosts, _ := redisClient.SInter("index:live").Result()

	for _, i := range hosts {
		// we now break at ':' and save the clusters piece
		s := strings.SplitN(i, ":", 2)
		if s[0] == queryCluster {
			// if the custer matches the query, throw the host in a tmp set
			redisClient.SAdd("tmp:cluster:index", s[1])
		}
	}
	// grab the set and delete
	clusters, _ := redisClient.SInter("tmp:cluster:index").Result()
	redisClient.Del("tmp:cluster:index")
	fmt.Fprintln(w, clusters)
}

func apiHostSpecHandler(w http.ResponseWriter, r *http.Request, redisClient *redis.Client) {
	cluster := strings.TrimSpace(r.FormValue("cluster"))
	host := strings.TrimSpace(r.FormValue("host"))

	ip, _ := redisClient.Get(fmt.Sprintf("cluster:%s:%s", cluster, host)).Result()

	fmt.Fprintln(w, ip)
}

func spawnClusterManager(cluster, hostname, ip, port string, redisClient *redis.Client) {
	// add in a connection drain redis entry cluster:<cluster_name>:<hostname> <drain time>
	//redisClient.Set(fmt.Sprintf("drain:%s:%s", cluster, hostname), config.Clustermanager.ConnectionDrain, 0).Err()
	connectionDrain := config.Clustermanager.ConnectionDrain

	if config.Clustermanager.ClientPingType == "port" {
		glogger.Cluster.Printf("spawning async cluster manager for %s:%s on port %s", cluster, hostname, port)
	} else {
		glogger.Cluster.Printf("spawning async cluster manager for %s:%s", cluster, hostname)
	}

	online := true
	for online {
		// TODO add logic for ICMP ping
		//if nsmanager.PingHost(ip) {
		healthCheck, err := nsmanager.HealthCheck(ip, port)
		if healthCheck {
			glogger.Debug.Printf("- %s:%s online", cluster, hostname)
		} else {
			if connectionDrain < (0 + int(config.Clustermanager.PingFreq)) {
				glogger.Debug.Printf("- %s:%s offline: error %s", cluster, hostname, err)
				online = false
				break
			}
			// print draining message if first shot
			if connectionDrain == config.Clustermanager.ConnectionDrain {
				glogger.Cluster.Printf("%s:%s listener draining", cluster, hostname)
			}
			connectionDrain = connectionDrain - int(config.Clustermanager.PingFreq)
		}
		// time between host pings
		time.Sleep(time.Second * config.Clustermanager.PingFreq)
	}
	glogger.Cluster.Printf("closing %s:%s listener", cluster, hostname)
	// remove the server entry, it is no longer online
	redisClient.Del(fmt.Sprintf("cluster:%s:%s", cluster, hostname))

	// remove the port entry, it is no longer online
	redisClient.Del(fmt.Sprintf("port:%s:%s", cluster, hostname))

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
		port, _ := redisClient.Get(fmt.Sprintf("port:%s:%s", cluster, hostname)).Result()
		// spawn cluster manager for host
		go spawnClusterManager(cluster, hostname, ip, port, redisClient)
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
