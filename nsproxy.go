package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"

	"github.com/miekg/dns"
)

/*
// =============================================================
// just a few tips when starting the server
// the linux network manager generally wont let you
// start the server on 53 when dnsmasq is already running.
// to demo the functionality of the server properly we need
// to stop dnsmasq first. To do this we can use 'ss -ant' to
// verify that port 53 is in use and 'lsof -i tcp:53' to make
// sure that dnsmasq is the culprit and find its pid. After
// the pid is found we kill it like normal with 'kill'. When
// we finish testing we should probably start dnsmasq again.
// we do this with systemctl start dnsmasq (on systemd machines)
// =============================================================
// we also need to point /etc/resolve.conf to our custom nameserver
// we can do this by changing the 'nameserver' entry to loopback:
// 127.0.0.1
// =============================================================
*/

/*
// =============================================================
// general strategy:
// This is super simple since we do not have
// A ton of entries. We should have a directory in the same path
// that the DNS server runs and inside that directory is a set of
// files that are formatted as follows:
//
// filename: FQDM
// content:  ipv4 address
//
// this will allow us to easily see if we have an entry and pack
// any new content into the response RR.
// =============================================================
*/
var defaultServer string

func main() {
	defaultServerAsk := flag.String("upstream", "8.8.8.8:53", "upstream nameserver")
	rawPort := flag.String("p", "53", "port to listen on")
	flag.Parse()
	defaultServer = *defaultServerAsk

	//format the string to be :port
	port := fmt.Sprint(":", *rawPort)

	udpServer := &dns.Server{Addr: port, Net: "udp"}
	tcpServer := &dns.Server{Addr: port, Net: "tcp"}
	println("started server on", *rawPort)
	// miekg/dns forces use to have a function for the handler, I'll submit a pr so he fixes it
	dns.HandleFunc(".", route)

	go func() {
		log.Fatal(udpServer.ListenAndServe())
	}()
	log.Fatal(tcpServer.ListenAndServe())
}

func route(w dns.ResponseWriter, req *dns.Msg) {
	proxy(defaultServer, w, req)
}

func proxy(addr string, w dns.ResponseWriter, req *dns.Msg) {
	hostname := req.Question[0].Name

	transport := "udp"
	if _, ok := w.RemoteAddr().(*net.TCPAddr); ok {
		transport = "tcp"
	}
	c := &dns.Client{Net: transport}
	resp, _, err := c.Exchange(req, addr)

	if err != nil {
		dns.HandleFailed(w, req)
		return
	}

	// we should be able to repack a custom answer if the fully qualified hostname
	// is found in our storage.
	//
	// if a record is found we parse the ipv4 address and build a new 'Answer' RR

	filepath := fmt.Sprintf("records/%s", hostname)

	if _, err := os.Stat(filepath); err == nil {
		//found a match; continue

		rr := new(dns.A)
		rr.Hdr = dns.RR_Header{Name: hostname, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 0}
		tmpaddr, _ := ioutil.ReadFile(filepath)
		addr := strings.TrimSuffix(string(tmpaddr), "\n")

		rr.A = net.ParseIP(addr)

		rep := new(dns.Msg)
		rep.SetReply(req)
		rep.Answer = append(rep.Answer, rr)

		println("serving", hostname, "from local record")
		w.WriteMsg(rep)
		return

	}

	println("serving", hostname, "from", defaultServer)
	w.WriteMsg(resp)
}
