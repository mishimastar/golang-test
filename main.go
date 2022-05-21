package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

var address = "127.0.0.1:port"

var Queues = struct {
	sync.RWMutex
	queue map[string](chan string)
}{queue: make(map[string](chan string))}

func main() {
	port_to_listen := flag.String("p", "80", "port to listen")
	flag.Parse()
	address := fmt.Sprintf("127.0.0.1:%s", *port_to_listen)
	http.HandleFunc("/", myHandler)
	fmt.Println("Listening on", address, "...")
	log.Fatal(http.ListenAndServe(address, nil))
}

func myHandler(w http.ResponseWriter, r *http.Request) {
	request_path := r.URL.Path
	switch r.Method {
	case "PUT":
		key_v := r.URL.Query().Get("v")
		if len(key_v) == 0 {
			w.WriteHeader(http.StatusBadRequest)
		} else {
			Queues.RLock()
			_, found := Queues.queue[request_path]
			Queues.RUnlock()
			if found {
				w.WriteHeader(http.StatusOK)
				go func() {
					Queues.queue[request_path] <- key_v
				}()
			} else {
				Queues.Lock()
				Queues.queue[request_path] = make(chan string)
				Queues.Unlock()
				go func() {
					Queues.queue[request_path] <- key_v
				}()
				w.WriteHeader(http.StatusOK)
			}

		}
	case "GET":
		key_timeout := r.URL.Query().Get("timeout")
		timeout, err := strconv.Atoi((key_timeout))
		switch {
		case len(key_timeout) == 0:
			timeout = 0
		case err != nil:
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		Queues.RLock()
		_, found := Queues.queue[request_path]
		Queues.RUnlock()
		if !found {
			Queues.Lock()
			Queues.queue[request_path] = make(chan string)
			Queues.Unlock()
		}
		select {
		case <-time.After(time.Duration(timeout) * time.Second):
			w.WriteHeader(http.StatusNotFound)
			return
		case answer := <-Queues.queue[request_path]:
			w.Write([]byte(answer))
			w.WriteHeader(http.StatusOK)
			return
		}

	default:
		w.WriteHeader(http.StatusBadRequest)
	}
}
