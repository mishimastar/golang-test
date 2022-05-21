package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
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

func cancelled(d chan struct{}) bool {
	select {
	case <-d:
		return true
	default:
		return false
	}
}

func remove[T comparable](list []T, unit T) []T {
	for i, item := range list {
		if item == unit {
			return append(list[:i], list[i+1:]...)
		}
	}
	return list
}

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
			PutQueues.Lock()
			_, found := PutQueues.queue[r.URL.Path]
			PutQueues.Unlock()
			if found {
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
		if len(key_timeout) == 0 || key_timeout == "0" {
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
		} else {
			timeout, err := strconv.Atoi((key_timeout))
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			pointer := &w
			GetQueues.Lock()
			_, found := GetQueues.queue[r.URL.Path]
			GetQueues.Unlock()
			if found {
				GetQueues.Lock()
				GetQueues.queue[r.URL.Path] = append(GetQueues.queue[r.URL.Path], pointer)
				GetQueues.Unlock()
			} else {
				GetQueues.Lock()
				GetQueues.queue[r.URL.Path] = []*http.ResponseWriter{pointer}
				GetQueues.Unlock()
			}
			sucsess := make(chan string)
			done := make(chan struct{})
			go func() {
				var current_get *http.ResponseWriter
				var path_found_put bool
				for {
					GetQueues.Lock()
					current_get = GetQueues.queue[r.URL.Path][0]
					GetQueues.Unlock()
					if current_get == pointer {
						PutQueues.Lock()
						_, path_found_put = PutQueues.queue[r.URL.Path]
						PutQueues.Unlock()
						if path_found_put {
							PutQueues.Lock()
							res := PutQueues.queue[r.URL.Path][0]
							PutQueues.queue[r.URL.Path] = PutQueues.queue[r.URL.Path][1:]
							if len(PutQueues.queue[r.URL.Path]) == 0 {
								delete(PutQueues.queue, r.URL.Path)
							}
							PutQueues.Unlock()
							GetQueues.Lock()
							GetQueues.queue[r.URL.Path] = GetQueues.queue[r.URL.Path][1:]
							if len(GetQueues.queue[r.URL.Path]) == 0 {
								delete(GetQueues.queue, r.URL.Path)
							}
							GetQueues.Unlock()

							sucsess <- res
							return
						}
					}
					if cancelled(done) {
						return
					}
				}
			}()
			fmt.Println(pointer, "started waiting for", r.URL.Path, "timeout =", timeout, " seconds")
			select {
			case <-time.After(time.Duration(timeout) * time.Second):
				close(done)
				GetQueues.Lock()
				GetQueues.queue[r.URL.Path] = remove(GetQueues.queue[r.URL.Path], pointer)
				if len(GetQueues.queue[r.URL.Path]) == 0 {
					delete(GetQueues.queue, r.URL.Path)
				}
				GetQueues.Unlock()
				w.WriteHeader(http.StatusNotFound)
			case answer := <-sucsess:
				w.Write([]byte(answer))
				w.WriteHeader(http.StatusOK)
				close(done)
				return
			}
		}
	default:
		w.WriteHeader(http.StatusBadRequest)
	}
}
