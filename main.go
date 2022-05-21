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

var GetQueues = struct {
	sync.Mutex
	queue map[string][]*http.ResponseWriter
}{queue: make(map[string][]*http.ResponseWriter)}

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
	case "GET":
		key_timeout := r.URL.Query().Get("timeout")
		if len(key_timeout) == 0 {
			GetQueues.Lock()
			_, found := GetQueues.queue[r.URL.Path]
			GetQueues.Unlock()
			if !found {
				PutQueues.Lock()
				_, found := PutQueues.queue[r.URL.Path]
				PutQueues.Unlock()
				if found {
					PutQueues.Lock()
					w.Write([]byte(PutQueues.queue[r.URL.Path][0]))
					PutQueues.queue[r.URL.Path] = PutQueues.queue[r.URL.Path][1:]
					if len(PutQueues.queue[r.URL.Path]) == 0 {
						delete(PutQueues.queue, r.URL.Path)
					}
					PutQueues.Unlock()
					w.WriteHeader(http.StatusOK)
				} else {
					w.WriteHeader(http.StatusNotFound)
				}
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}
	}
}
