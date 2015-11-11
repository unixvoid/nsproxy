package main

import (
	"flag"
	"gonsman"
	"os"

	"github.com/unixvoid/nsproxy/nsproxymanager.go"
)

/*
// =============================================
// this is the entry manager for the goNSproxy.
// the design goals are as follows:
//   - list entries and their attributes
//   - add entries
//   - remove entries
//   - modify entries
//
// I would also like to add a listener on a
// different port so we can manage this thing
// while it is deployed. I think we should use
// gorilla/mux and just listen on 8054 or something
// and a simple rest request to manage entries
// =============================================
*/

func main() {
	list := flag.Bool("l", false, "list records")
	addReq := flag.String("add", "", "add record: <domain name> <ipv4 addr>")
	remReq := flag.String("rm", "", "remove a record: <domain name>")
	modReq := flag.String("modify", "", "modify record: <domain name> <ipv4 addr>")

	flag.Parse()

	// first thing to do is make a 'records/' dir
	// if it does not exist
	_, err := os.Stat("records/")
	if err != nil {
		os.Mkdir("records/", 0755)
	}

	if *list {
		nsproxymanager.listEntries()
	}

	if *addReq != "" {
		dn := *addReq
		// the extra argument is the ip
		ip := flag.Arg(0)
		nsproxymanager.addEntry(dn, ip)
	}

	if *modReq != "" {
		dn := *modReq
		// the extra argument is the ip
		ip := flag.Arg(0)
		nsproxymanager.addEntry(dn, ip)
	}

	if *remReq != "" {
		rm := *remReq
		nsproxymanager.rmEntry(rm)
	}
}
