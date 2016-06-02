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

	// we should be able to repack a custom answer if the fully qualified hostname
	// is found in our storage.
	//
	// if a record is found we parse the ipv4 address and build a new 'Answer' RR

	glogger.Debug.Printf("querying dns:a:%s", hostname)
	lookup, err := nsmanager.Query("dns", "a", hostname, redisClient)

	if err == nil {
		//found a match; continue

		rr := new(dns.A)
		rr.Hdr = dns.RR_Header{Name: hostname, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 0}
		addr := strings.TrimSuffix(lookup, "\n")

		rr.A = net.ParseIP(addr)

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
