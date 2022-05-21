package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
)

var address = "127.0.0.1:port"

func main() {
	port_to_listen := flag.String("p", "80", "port to listen")
	flag.Parse()
	address = strings.ReplaceAll(address, "port", *port_to_listen)
	// fmt.Println("Inintial commit")
	// router := http.NewServeMux
	http.HandleFunc("/", myHandler)
	fmt.Println("Listening on", address, "...")
	log.Fatal(http.ListenAndServe(address, nil))
}

func myHandler(w http.ResponseWriter, r *http.Request) {

}
