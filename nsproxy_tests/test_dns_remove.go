package main

import (
	"github.com/unixvoid/glogger"
)

func testDnsRemove(hostUrl *string) {

	// remove DNS entry
	glogger.Info.Println("Testing :: POST to /dns/rm A type")
	// domain not set
	returnVal := twoKeyPostEndpoint(*hostUrl, "/dns/rm", "", "test.domain", "dnstype", "a")
	checkResponse("Domain not set", 400, returnVal)
	// correct
	returnVal = twoKeyPostEndpoint(*hostUrl, "/dns/rm", "domain", "test.domain", "dnstype", "a")
	checkResponse("Correct", 200, returnVal)
	glogger.Info.Println("")

	glogger.Info.Println("Testing :: POST to /dns/rm cname type")
	// domain not set
	returnVal = twoKeyPostEndpoint(*hostUrl, "/dns/rm", "", "test.domain", "dnstype", "cname")
	checkResponse("Domain not set", 400, returnVal)
	// correct
	returnVal = twoKeyPostEndpoint(*hostUrl, "/dns/rm", "domain", "test.domain", "dnstype", "cname")
	checkResponse("Correct", 200, returnVal)
	glogger.Info.Println("")

	glogger.Info.Println("Testing :: POST to /dns/rm aaaa type")
	// domain not set
	returnVal = twoKeyPostEndpoint(*hostUrl, "/dns/rm", "", "test.domain", "dnstype", "aaaa")
	checkResponse("Domain not set", 400, returnVal)
	// correct
	returnVal = twoKeyPostEndpoint(*hostUrl, "/dns/rm", "domain", "test.domain", "dnstype", "aaaa")
	checkResponse("Correct", 200, returnVal)
	glogger.Info.Println("")

	glogger.Info.Println("Testing :: POST to /dns/rm type unset")
	// correct
	returnVal = twoKeyPostEndpoint(*hostUrl, "/dns/rm", "domain", "test.domain", "", "")
	checkResponse("Dnstype unset", 200, returnVal)
}
