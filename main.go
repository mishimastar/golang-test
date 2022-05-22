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

type Queues struct {
	mu    sync.RWMutex
	queue map[string](chan string)
}

func (q *Queues) pathInQueue(path string) {
	q.mu.RLock()
	_, found := q.queue[path]
	q.mu.RUnlock()
	if !found {
		q.mu.Lock()
		q.queue[path] = make(chan string)
		q.mu.Unlock()
	}
}

var Queue = Queues{queue: make(map[string](chan string))}

func main() {
	// queue := Queues{queue: make(map[string](chan string))}
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
			Queue.pathInQueue(request_path)
			w.WriteHeader(http.StatusOK)
			go func() {
				Queue.queue[request_path] <- key_v
			}()

		}
	case "GET":
		key_timeout := r.URL.Query().Get("timeout")
		timeout, err := strconv.Atoi(key_timeout)
		switch {
		case len(key_timeout) == 0:
			timeout = 0
		case err != nil:
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		Queue.pathInQueue(request_path)
		select {
		case <-time.After(time.Duration(timeout) * time.Second):
			w.WriteHeader(http.StatusNotFound)
			return
		case answer := <-Queue.queue[request_path]:
			w.Write([]byte(answer))
			w.WriteHeader(http.StatusOK)
			return
		}

	default:
		w.WriteHeader(http.StatusBadRequest)
	}
}
