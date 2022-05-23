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

func (q *Queues) pathInQueue(path string) chan string {
	q.mu.Lock()
	defer q.mu.Unlock()
	_, found := q.queue[path]
	if !found {
		q.queue[path] = make(chan string)
	}
	return q.queue[path]
}

var Queue = Queues{queue: make(map[string](chan string))}

func main() {
	// queue := Queues{queue: make(map[string](chan string))}
	portToListen := flag.String("p", "80", "port to listen")
	flag.Parse()
	address := fmt.Sprintf("127.0.0.1:%s", *portToListen)
	http.HandleFunc("/", myHandler)
	fmt.Println("Listening on", address, "...")
	log.Fatal(http.ListenAndServe(address, nil))
}

func myHandler(w http.ResponseWriter, r *http.Request) {
	requestPath := r.URL.Path
	switch r.Method {
	case "PUT":
		keyV := r.URL.Query().Get("v")
		if len(keyV) == 0 {
			w.WriteHeader(http.StatusBadRequest)
		} else {
			w.WriteHeader(http.StatusOK)
			go func() {
				Queue.pathInQueue(requestPath) <- keyV
			}()

		}
	case "GET":
		keyTimeout := r.URL.Query().Get("timeout")
		timeout, err := strconv.Atoi(keyTimeout)
		switch {
		case len(keyTimeout) == 0:
			timeout = 0
		case err != nil:
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		select {
		case <-time.After(time.Duration(timeout) * time.Second):
			w.WriteHeader(http.StatusNotFound)
			return
		case answer := <-Queue.pathInQueue(requestPath):
			w.Write([]byte(answer))
			w.WriteHeader(http.StatusOK)
			return
		}

	default:
		w.WriteHeader(http.StatusBadRequest)
	}
}
