package main

import (
	"github.com/unixvoid/glogger"
)

func testDnsAdd(hostUrl *string) {

	// add DNS entry
	glogger.Info.Println("Testing :: POST to /dns A type")
	// correct
	returnVal := twoKeyPostEndpoint(*hostUrl, "/dns", "domain", "test.domain", "value", "127.0.0.1")
	checkResponse("Correct", 200, returnVal)
	// domain not set
	returnVal = twoKeyPostEndpoint(*hostUrl, "/dns", "", "test.domain", "value", "127.0.0.1")
	checkResponse("Domain not set", 400, returnVal)
	// value not set
	returnVal = twoKeyPostEndpoint(*hostUrl, "/dns", "domain", "test.domain", "", "127.0.0.1")
	checkResponse("Value not set", 400, returnVal)
	glogger.Info.Println("")

	glogger.Info.Println("Testing :: POST to /dns cname type")
	// correct
	returnVal = threeKeyPostEndpoint(*hostUrl, "/dns", "domain", "test.domain", "value", "test.domain.lb", "dnstype", "cname")
	checkResponse("Correct", 200, returnVal)
	// domain not set
	returnVal = threeKeyPostEndpoint(*hostUrl, "/dns", "", "test.domain", "value", "test.domain.lb", "dnstype", "cname")
	checkResponse("Domain not set", 400, returnVal)
	// value not set
	returnVal = threeKeyPostEndpoint(*hostUrl, "/dns", "domain", "test.domain", "", "test.domain.lb", "dnstype", "cname")
	checkResponse("Value not set", 400, returnVal)
	// unsupported type
	returnVal = threeKeyPostEndpoint(*hostUrl, "/dns", "domain", "test.domain", "value", "test.domain.lb", "dnstype", "cnames")
	checkResponse("Unsupported dnstype", 400, returnVal)
	glogger.Info.Println("")

	glogger.Info.Println("Testing :: POST to /dns aaaa type")
	// correct
	returnVal = threeKeyPostEndpoint(*hostUrl, "/dns", "domain", "test.domain", "value", "::1", "dnstype", "aaaa")
	checkResponse("Correct", 200, returnVal)
	// domain not set
	returnVal = threeKeyPostEndpoint(*hostUrl, "/dns", "", "test.domain", "value", "::1", "dnstype", "aaaa")
	checkResponse("Domain not set", 400, returnVal)
	// value not set
	returnVal = threeKeyPostEndpoint(*hostUrl, "/dns", "domain", "test.domain", "", "::1", "dnstype", "aaaa")
	checkResponse("Value not set", 400, returnVal)
	// unsupported type
	returnVal = threeKeyPostEndpoint(*hostUrl, "/dns", "domain", "test.domain", "value", "::1", "dnstype", "aaab")
	checkResponse("Unsupported dnstype", 400, returnVal)
}
