package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
)

var address = "127.0.0.1:port"

var PutQueues = struct {
	sync.Mutex
	queue map[string][]string
}{queue: make(map[string][]string)}

func main() {
	port_to_listen := flag.String("p", "80", "port to listen")
	flag.Parse()
	address = strings.ReplaceAll(address, "port", *port_to_listen)
	http.HandleFunc("/", myHandler)
	fmt.Println("Listening on", address, "...")
	log.Fatal(http.ListenAndServe(address, nil))
}

func myHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "PUT":
		key_v := r.URL.Query().Get("v")
		if len(key_v) == 0 {
			w.WriteHeader(http.StatusBadRequest)
		} else {
			if _, found := PutQueues.queue[r.URL.Path]; found {
				w.WriteHeader(http.StatusOK)
				PutQueues.Lock()
				PutQueues.queue[r.URL.Path] = append(PutQueues.queue[r.URL.Path], key_v)
				fmt.Println(PutQueues.queue, "appended", key_v, "to", r.URL.Path)
				PutQueues.Unlock()
			} else {
				PutQueues.Lock()
				PutQueues.queue[r.URL.Path] = []string{key_v}
				fmt.Println(PutQueues.queue, "appended", key_v, "to created", r.URL.Path)
				PutQueues.Unlock()
			}

		}
	}
}
