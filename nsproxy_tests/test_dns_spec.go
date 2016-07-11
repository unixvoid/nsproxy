package main

import (
	"github.com/unixvoid/glogger"
)

func testDnsSpec(hostUrl *string) {

	// get dns info on test.domain
	glogger.Info.Println("Testing :: POST to /dnsspec A type")
	// correct
	body, returnVal := twoKeyPostReturnEndpoint(*hostUrl, "/dnsspec", "domain", "test.domain", "dnstype", "a")
	checkResponse("Correct", 200, returnVal)
	glogger.Info.Printf("/dnsspec GET reponds: %s", body)
	glogger.Info.Println("")

	glogger.Info.Println("Testing :: POST to /dnsspec AAAA type")
	// correct
	body, returnVal = twoKeyPostReturnEndpoint(*hostUrl, "/dnsspec", "domain", "test.domain", "dnstype", "aaaa")
	checkResponse("Correct", 200, returnVal)
	glogger.Info.Printf("/dnsspec GET reponds: %s", body)
	glogger.Info.Println("")

	glogger.Info.Println("Testing :: POST to /dnsspec CNAME type")
	// correct
	body, returnVal = twoKeyPostReturnEndpoint(*hostUrl, "/dnsspec", "domain", "test.domain", "dnstype", "cname")
	checkResponse("Correct", 200, returnVal)
	glogger.Info.Printf("/dnsspec GET reponds: %s", body)
	glogger.Info.Println("")

	glogger.Info.Println("Testing :: POST to /dnsspec no type")
	// correct
	body, returnVal = twoKeyPostReturnEndpoint(*hostUrl, "/dnsspec", "domain", "test.domain", "dnstype", "a")
	checkResponse("Correct", 200, returnVal)
	glogger.Info.Printf("/dnsspec GET reponds: %s", body)
	glogger.Info.Println("")
}
