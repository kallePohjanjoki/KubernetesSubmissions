package main

import (
	"crypto/rand"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

var randomString string

func generateRandomString() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func main() {
	randomString = generateRandomString()

	go func() {
		for {
			timestamp := time.Now().UTC().Format(time.RFC3339Nano)
			fmt.Printf("%s: %s\n", timestamp, randomString)
			time.Sleep(5 * time.Second)
		}
	}()

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		timestamp := time.Now().UTC().Format(time.RFC3339Nano)
		fmt.Fprintf(w, "%s: %s\n", timestamp, randomString)
	})

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
