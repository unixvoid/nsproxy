package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
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
		listEntries()
	}

	if *addReq != "" {
		dn := *addReq
		// the extra argument is the ip
		ip := flag.Arg(0)
		addEntry(dn, ip)
	}

	if *modReq != "" {
		dn := *modReq
		// the extra argument is the ip
		ip := flag.Arg(0)
		addEntry(dn, ip)
	}

	if *remReq != "" {
		rm := *remReq
		rmEntry(rm)
	}
}

func listEntries() {
	// list records
	println("-------------------------------------------")
	files, _ := ioutil.ReadDir("records/")
	for _, f := range files {
		//print out filename followed by record
		filepath := fmt.Sprintf("records/%s", f.Name())
		cont, _ := ioutil.ReadFile(filepath)
		//strip out trailing '.' from FQDM
		formName := f.Name()[:len(f.Name())-1]
		fmt.Printf("%-25s: %s", formName, cont)
	}
	println("-------------------------------------------")
	return
}

func addEntry(dn, ip string) {
	// add record
	// more useable when the newline is appened to the end
	ip = fmt.Sprintf("%s\n", ip)

	// if the domain is not FQ, FQ it
	if string(dn[len(dn)-1]) != "." {
		//println("not fully qualified")
		dn = fmt.Sprintf("%s.", dn)
	}
	// filepath is 'records/fqdn'
	path := fmt.Sprintf("records/%s", dn)

	// write record
	content := []byte(ip)
	ioutil.WriteFile(path, content, 0644)
}

func rmEntry(rm string) {
	// delete record
	// try to remove it if it exist, otherwise it doesn't
	// filepath is 'records/<rm>.', need to append a .
	path := fmt.Sprintf("records/%s.", rm)

	err := os.Remove(path)
	if err != nil {
		fmt.Println(err)
	}
}
