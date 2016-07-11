package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/unixvoid/glogger"
)

func main() {
	// parse flags
	logLevel := flag.String("loglevel", "debug", "Level of logging {'debug', 'info', 'cluster', 'error'}")
	hostUrl := flag.String("host", "http://localhost:8080", "Endpoint to make requests")
	flag.Parse()

	// init logger
	if *logLevel == "debug" {
		glogger.LogInit(os.Stdout, os.Stdout, os.Stdout, os.Stderr)
	} else if *logLevel == "cluster" {
		glogger.LogInit(os.Stdout, os.Stdout, ioutil.Discard, os.Stderr)
	} else if *logLevel == "info" {
		glogger.LogInit(os.Stdout, ioutil.Discard, ioutil.Discard, os.Stderr)
	} else {
		glogger.LogInit(ioutil.Discard, ioutil.Discard, ioutil.Discard, os.Stderr)
	}

	glogger.Info.Println("====================== test_dns_add.go ======================")
	testDnsAdd(hostUrl)
	glogger.Info.Println("====================== endpoint completed ===================")
	glogger.Info.Println("")

	glogger.Info.Println("====================== test_dns_get.go ======================")
	testDnsGet(hostUrl)
	glogger.Info.Println("====================== endpoint completed ===================")
	glogger.Info.Println("")

	glogger.Info.Println("====================== test_dns_spec.go =====================")
	testDnsSpec(hostUrl)
	glogger.Info.Println("====================== endpoint completed ===================")
	glogger.Info.Println("")

	glogger.Info.Println("====================== test_dns_remove.go ===================")
	testDnsRemove(hostUrl)
	glogger.Info.Println("====================== endpoint completed ===================")
	glogger.Info.Println("")
}

func twoKeyPostEndpoint(hostUrl, endpoint, firstKey, firstValue, secondKey, secondValue string) int {
	postData := url.Values{}
	postData.Set(firstKey, firstValue)
	postData.Add(secondKey, secondValue)

	client := &http.Client{}
	req, _ := http.NewRequest("POST", fmt.Sprintf("%s%s", hostUrl, endpoint), strings.NewReader(postData.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		glogger.Error.Println(err)
	}

	return resp.StatusCode
}

func threeKeyPostEndpoint(hostUrl, endpoint, firstKey, firstValue, secondKey, secondValue, thirdKey, thirdValue string) int {
	postData := url.Values{}
	postData.Set(firstKey, firstValue)
	postData.Add(secondKey, secondValue)
	postData.Add(thirdKey, thirdValue)

	client := &http.Client{}
	req, _ := http.NewRequest("POST", fmt.Sprintf("%s%s", hostUrl, endpoint), strings.NewReader(postData.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		glogger.Error.Println(err)
	}

	return resp.StatusCode
}

func twoKeyPostReturnEndpoint(hostUrl, endpoint, firstKey, firstValue, secondKey, secondValue string) (string, int) {
	postData := url.Values{}
	postData.Set(firstKey, firstValue)
	postData.Add(secondKey, secondValue)

	client := &http.Client{}
	req, _ := http.NewRequest("POST", fmt.Sprintf("%s%s", hostUrl, endpoint), strings.NewReader(postData.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		glogger.Error.Println(err)
	}
	defer resp.Body.Close()
	bodyMsg, _ := ioutil.ReadAll(resp.Body)

	return string(bodyMsg), resp.StatusCode
}

func checkResponse(prefix string, wantedResponse, respCode int) {
	if respCode == wantedResponse {
		glogger.Info.Printf("%s :: Test passed, got %v", prefix, respCode)
	} else {
		glogger.Error.Printf("%s :: Expected %v, got %v", prefix, wantedResponse, respCode)
	}
}

func getEndpoint(hostUrl, endpoint string) (string, int) {
	resp, err := http.Get(fmt.Sprintf("%s%s", hostUrl, endpoint))
	if err != nil {
		glogger.Error.Println(err)
	}
	defer resp.Body.Close()
	bodyMsg, _ := ioutil.ReadAll(resp.Body)
	return string(bodyMsg), resp.StatusCode
}
