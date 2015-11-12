package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
	"github.com/unixvoid/nsproxy/nsmanager"
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
	rawPort := flag.String("p", "8054", "port to listen on")
	flag.Parse()
	port := fmt.Sprint(":", *rawPort)
	// first thing to do is make a 'records/' dir
	// if it does not exist
	_, err := os.Stat("records/")
	if err != nil {
		os.Mkdir("records/", 0755)
	}

	router := mux.NewRouter()
	router.HandleFunc("/!{command}", dynamichandler).Methods("GET")
	println("running on port", *rawPort)
	log.Fatal(http.ListenAndServe(port, router))
}

func dynamichandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	commArr := strings.SplitN(vars["command"], " ", 3)
	command := commArr[0]

	switch command {
	case "list":
		fmt.Fprintln(w, "-------------------------------------------")
		files, _ := ioutil.ReadDir("records/")
		for _, f := range files {
			//print out filename followed by record
			filepath := fmt.Sprintf("records/%s", f.Name())
			cont, _ := ioutil.ReadFile(filepath)
			//strip out trailing '.' from FQDM
			formName := f.Name()[:len(f.Name())-1]
			fmt.Fprintf(w, "%-25s: %s", formName, cont)
		}
		fmt.Fprintf(w, "-------------------------------------------")
		return

	case "add":
		dn, ip := commArr[1], commArr[2]
		nsmanager.AddEntry(dn, ip)
	case "rm":
		rm := commArr[1]
		nsmanager.RmEntry(rm)
	case "modify":
		dn, ip := commArr[1], commArr[2]
		nsmanager.AddEntry(dn, ip)
	default:
		fmt.Fprintf(w, "not a valid command.")
	}
}
