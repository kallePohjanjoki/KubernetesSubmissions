package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
)

type Todo struct {
	ID      int    `json:"id"`
	Content string `json:"content"`
}

var (
	mu     sync.Mutex
	todos  []Todo
	nextID = 1
)

func todosHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(todos)

	case http.MethodPost:
		var body struct {
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}

		content := strings.TrimSpace(body.Content)
		if content == "" {
			http.Error(w, "Content cannot be empty", http.StatusBadRequest)
			return
		}
		if len(content) > 140 {
			http.Error(w, "Content must be 140 characters or fewer", http.StatusBadRequest)
			return
		}

		mu.Lock()
		newTodo := Todo{ID: nextID, Content: content}
		nextID++
		todos = append(todos, newTodo)
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(newTodo)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	http.HandleFunc("/todos", todosHandler)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
