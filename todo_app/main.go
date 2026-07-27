package main

import (
	"encoding/base64"
	"encoding/json"
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

var httpClient = &http.Client{
	Timeout: 2 * time.Second,
}

type Todo struct {
	ID      int    `json:"id"`
	Content string `json:"content"`
}

func imagePath() string {
	p := os.Getenv("IMAGE_PATH")

	return p
}

// fetchAndSaveImage downloads a fresh image from Lorem Picsum and writes it to disk.
func fetchAndSaveImage(path string) error {
	imageSourceURL := os.Getenv("IMAGE_SOURCE_URL")
	resp, err := http.Get(imageSourceURL)

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

func fetchTodos() ([]Todo, error) {
	todoBackendURL := os.Getenv("TODO_BACKEND_URL")
	if todoBackendURL == "" {
		todoBackendURL = "http://todobackend-svc:2345/todos"
	}

	resp, err := httpClient.Get(todoBackendURL)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var todos []Todo
	if err := json.NewDecoder(resp.Body).Decode(&todos); err != nil {
		return nil, err
	}
	return todos, nil
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

	todos, err := fetchTodos()
	if err != nil {
		log.Printf("failed to fetch todos: %v", err)
		todos = []Todo{}
	}

	tmpl, err := template.ParseFiles("index.html")
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	data := struct {
		ImageBase64 string
		Todos       []Todo
	}{
		ImageBase64: base64.StdEncoding.EncodeToString(imageBytes),
		Todos:       todos,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("failed to render page: %v", err)
	}
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	http.HandleFunc("/", indexHandler)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
