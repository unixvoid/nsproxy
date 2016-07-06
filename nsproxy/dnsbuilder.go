package main

import (
	"net"
	"strings"

	"github.com/miekg/dns"
	"github.com/unixvoid/nsproxy/pkg/nslog"
	"github.com/unixvoid/nsproxy/pkg/nsmanager"
	"gopkg.in/redis.v3"
)

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

		nslog.Debug.Println("serving", hostname, "from local record")
		w.WriteMsg(rep)
		return
	} else {
		// domain does not exist, return dns error
		nslog.Debug.Println("serving", hostname, "from", config.Upstreamdns.Server)
		w.WriteMsg(resp)
	}
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
