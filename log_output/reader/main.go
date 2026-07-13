package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

func main() {
	port := os.Getenv("PORT")

	statusFilePath := os.Getenv("FILE_PATH")
	if statusFilePath == "" {
		statusFilePath = "/shared/status.txt"
	}

	counterFilePath := os.Getenv("COUNTER_FILE_PATH")
	if counterFilePath == "" {
		counterFilePath = "/pingpong-data/pingpong_counter.txt"
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		statusContent, err := os.ReadFile(statusFilePath)
		if err != nil {
			http.Error(w, "Status not available yet", http.StatusServiceUnavailable)
			return
		}

		counterContent, err := os.ReadFile(counterFilePath)
		counterValue := "0"
		if err == nil {
			counterValue = strings.TrimSpace(string(counterContent))
		}

		line := strings.TrimRight(string(statusContent), "\n")
		fmt.Fprintf(w, "%s. Ping / pongs: %s\n", line, counterValue)
	})

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
