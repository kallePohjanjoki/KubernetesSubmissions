package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
)

var mu sync.Mutex

func readCounter(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	n, err := strconv.Atoi(string(data))
	if err != nil {
		return 0
	}
	return n
}

func writeCounter(path string, n int) error {
	return os.WriteFile(path, []byte(strconv.Itoa(n)), 0644)
}

func main() {
	port := os.Getenv("PORT")

	filePath := os.Getenv("FILE_PATH")
	if filePath == "" {
		filePath = "/shared/pingpong_counter.txt"
	}

	http.HandleFunc("/pingpong", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		mu.Lock()
		current := readCounter(filePath)
		if err := writeCounter(filePath, current+1); err != nil {
			mu.Unlock()
			http.Error(w, "Failed to write counter", http.StatusInternalServerError)
			return
		}
		mu.Unlock()

		fmt.Fprintf(w, "pong %d\n", current)
	})

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
