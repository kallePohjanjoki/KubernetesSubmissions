package main

import (
	"encoding/base64"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

const cacheDuration = 10 * time.Minute

var (
	imageMu    sync.Mutex
	refreshing bool
)

func imagePath() string {
	p := "/images/image.jpg"

	return p
}

// fetchAndSaveImage downloads a fresh image from Lorem Picsum and writes it to disk.
func fetchAndSaveImage(path string) error {
	resp, err := http.Get("https://picsum.photos/1200")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status from picsum: %d", resp.StatusCode)
	}

	tmpPath := path + ".tmp"
	out, err := os.Create(tmpPath)
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, resp.Body); err != nil {
		out.Close()
		os.Remove(tmpPath)
		return err
	}
	out.Close()

	return os.Rename(tmpPath, path)
}

func refreshInBackground(path string) {
	imageMu.Lock()
	if refreshing {
		imageMu.Unlock()
		return
	}
	refreshing = true
	imageMu.Unlock()

	go func() {
		defer func() {
			imageMu.Lock()
			refreshing = false
			imageMu.Unlock()
		}()

		if err := fetchAndSaveImage(path); err != nil {
			log.Printf("failed to refresh image: %v", err)
		}
	}()
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := imagePath()
	info, err := os.Stat(path)

	if err != nil {
		if fetchErr := fetchAndSaveImage(path); fetchErr != nil {
			http.Error(w, "Could not fetch image", http.StatusBadGateway)
			return
		}
		info, err = os.Stat(path)
		if err != nil {
			http.Error(w, "Image not available", http.StatusInternalServerError)
			return
		}
	} else if time.Since(info.ModTime()) > cacheDuration {
		refreshInBackground(path)
	}

	imageBytes, err := os.ReadFile(path)
	if err != nil {
		http.Error(w, "Image not available", http.StatusInternalServerError)
		return
	}

	tmpl, err := template.ParseFiles("index.html")
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	data := struct {
		ImageBase64 string
	}{
		ImageBase64: base64.StdEncoding.EncodeToString(imageBytes),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("failed to render page: %v", err)
	}
}

func main() {
	port := os.Getenv("PORT")

	http.HandleFunc("/", indexHandler)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
