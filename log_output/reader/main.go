package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")

	filePath := os.Getenv("FILE_PATH")
	if filePath == "" {
		filePath = "/shared/status.txt"
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		content, err := os.ReadFile(filePath)
		if err != nil {
			http.Error(w, "Status not available yet", http.StatusServiceUnavailable)
			return
		}
		w.Write(content)
	})

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
