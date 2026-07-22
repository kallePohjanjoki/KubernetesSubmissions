package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
)

var (
	counter int
	mu      sync.Mutex
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	http.HandleFunc("/pingpong", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		mu.Lock()
		current := counter
		counter++
		mu.Unlock()

		fmt.Fprintf(w, "pong %d\n", current)
	})

	http.HandleFunc("/pings", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		mu.Lock()
		current := counter
		mu.Unlock()

		fmt.Fprint(w, strconv.Itoa(current))
	})

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
