package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var httpClient = &http.Client{
	Timeout: 2 * time.Second,
}

func main() {
	port := os.Getenv("PORT")

	statusFilePath := "/shared/status.txt"
	infoFilePath := "/config/information.txt"

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		statusContent, err := os.ReadFile(statusFilePath)
		if err != nil {
			http.Error(w, "Status not available yet", http.StatusServiceUnavailable)
			return
		}

		counterValue := "0"
		resp, err := httpClient.Get("http://pingpong-svc.default.svc.cluster.local:2347/pings")
		if err == nil {
			defer resp.Body.Close()
			body, readErr := io.ReadAll(resp.Body)
			if readErr == nil {
				counterValue = strings.TrimSpace(string(body))
			}
		} else {
			log.Printf("failed to reach ping-pong app: %v", err)
		}

		fileContent, err := os.ReadFile(infoFilePath)
		fileText := "not available"
		if err == nil {
			fileText = strings.TrimSpace(string(fileContent))
		} else {
			log.Printf("failed to read info file: %v", err)
		}

		message := os.Getenv("MESSAGE")

		line := strings.TrimRight(string(statusContent), "\n")

		fmt.Fprintf(w, "file content: %s\n", fileText)
		fmt.Fprintf(w, "env variable: MESSAGE=%s\n", message)
		fmt.Fprintf(w, "%s. Ping / Pongs: %s\n", line, counterValue)
	})

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
