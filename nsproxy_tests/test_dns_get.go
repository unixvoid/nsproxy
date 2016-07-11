package main

import (
	"github.com/unixvoid/glogger"
)

func testDnsGet(hostUrl *string) {
	body, returnVal := getEndpoint(*hostUrl, "/dns")
	checkResponse("Get /dns", 200, returnVal)
	glogger.Info.Printf("/dns GET responds: %s", body)
}
