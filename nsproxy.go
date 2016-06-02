package main

import (
	"fmt"
	"gopkg.in/gcfg.v1"
	"io/ioutil"
	"log"
	"net"
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

	_, err = os.Stat("records/")
	if err != nil {
		os.Mkdir("records/", 0755)
	}

	//format the string to be :port
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
	//glogger.Debug.Println("---------------------------------------------------------")
	glogger.Debug.Printf("ID :: %v", req.MsgHdr.Id)
	//glogger.Debug.Printf("NS :: %v", req.Ns)
	//glogger.Debug.Printf("Header :: %v", req.MsgHdr)
	//glogger.Debug.Printf("Compress :: %v", req.Compress)
	glogger.Debug.Printf("Question :: %v", req.Question[0].Qtype)
	//glogger.Debug.Printf("Answer :: %v", req.Answer)
	//glogger.Debug.Printf("Extra :: %v", req.Extra)
	//glogger.Debug.Println("---------------------------------------------------------")
	//glogger.Debug.Printf("Req :: %v", req)

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

	// we should be able to repack a custom answer if the fully qualified hostname
	// is found in our storage.
	//
	// if a record is found we parse the ipv4 address and build a new 'Answer' RR
	//var dnsType uint16
	//var redisType string

	switch req.Question[0].Qtype {
	case 1:
		//dnsType = dns.TypeA
		//redisType = "a"
		aBuilder(w, req, resp, hostname, redisClient)
		break
	case 5:
		// https://github.com/miekg/dns/blob/master/types.go#L240
		//dnsType = dns.TypeCNAME
		//redisType = "cname"
		cnameBuilder(w, req, resp, hostname, redisClient)
		break
	case 28:
		// https://github.com/miekg/dns/blob/master/types.go#L632
		//dnsType = dns.TypeAAAA
		//redisType = "aaaa"
		aaaaBuilder(w, req, resp, hostname, redisClient)
		break
	default:
		//dnsType = dns.TypeA
		//redisType = "a"
		aBuilder(w, req, resp, hostname, redisClient)
		break
	}
}

func aBuilder(w dns.ResponseWriter, req, resp *dns.Msg, hostname string, redisClient *redis.Client) {
	glogger.Debug.Printf("querying dns:a:%s", hostname)
	lookup, err := nsmanager.Query("dns", "a", hostname, redisClient)

	if err == nil {
		// found a match; continue

		// craft the A record response
		rr := new(dns.A)
		rr.Hdr = dns.RR_Header{Name: hostname, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: config.Dns.Ttl}
		addr := strings.TrimSuffix(lookup, "\n")
		rr.A = net.ParseIP(addr)

		// create the response and append the crafted a portion to it
		rep := new(dns.Msg)
		rep.SetReply(req)
		rep.Answer = append(rep.Answer, rr)

		glogger.Debug.Println("serving", hostname, "from local record")
		w.WriteMsg(rep)
		return
	}

	glogger.Debug.Println("serving", hostname, "from", config.Upstreamdns.Server)
	w.WriteMsg(resp)
}

func aaaaBuilder(w dns.ResponseWriter, req, resp *dns.Msg, hostname string, redisClient *redis.Client) {
	glogger.Debug.Printf("querying dns:aaaa:%s", hostname)
	lookup, err := nsmanager.Query("dns", "aaaa", hostname, redisClient)

	if err == nil {
		// found a match; continue

		// craft the A record response
		rr := new(dns.AAAA)
		rr.Hdr = dns.RR_Header{Name: hostname, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: config.Dns.Ttl}
		addr := strings.TrimSuffix(lookup, "\n")
		rr.AAAA = net.ParseIP(addr)

		// create the response and append the crafted a portion to it
		rep := new(dns.Msg)
		rep.SetReply(req)
		rep.Answer = append(rep.Answer, rr)

		glogger.Debug.Println("serving", hostname, "from local record")
		w.WriteMsg(rep)
		return
	}

	glogger.Debug.Println("serving", hostname, "from", config.Upstreamdns.Server)
	w.WriteMsg(resp)
}

func cnameBuilder(w dns.ResponseWriter, req, resp *dns.Msg, hostname string, redisClient *redis.Client) {
	glogger.Debug.Printf("querying dns:cname:%s", hostname)
	lookup, err := nsmanager.Query("dns", "cname", hostname, redisClient)
	glogger.Debug.Printf("lookup is: %s", lookup)

	if err == nil {
		// found a match; continue

		// craft the A record response
		rr := new(dns.CNAME)
		rr.Hdr = dns.RR_Header{Name: hostname, Rrtype: dns.TypeCNAME, Class: dns.ClassINET, Ttl: config.Dns.Ttl}
		rr.Target = lookup

		// create the response and append the crafted a portion to it
		rep := new(dns.Msg)
		rep.SetReply(req)
		rep.Answer = append(rep.Answer, rr)

		glogger.Debug.Println("serving", hostname, "from local record")
		w.WriteMsg(rep)
		return
	}

	glogger.Debug.Println("serving", hostname, "from", config.Upstreamdns.Server)
	w.WriteMsg(resp)
}
