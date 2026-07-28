package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

type Todo struct {
	ID      int    `json:"id"`
	Content string `json:"content"`
}

var db *sql.DB

func connectWithRetry() (*sql.DB, error) {
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")

	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname,
	)

	var conn *sql.DB
	var err error

	for attempt := 1; attempt <= 20; attempt++ {
		conn, err = sql.Open("postgres", connStr)
		if err == nil {
			if pingErr := conn.Ping(); pingErr == nil {
				return conn, nil
			} else {
				err = pingErr
			}
		}
		log.Printf("database not ready yet (attempt %d/20): %v", attempt, err)
		time.Sleep(3 * time.Second)
	}

	return nil, err
}

func ensureSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS todos (
			id SERIAL PRIMARY KEY,
			content TEXT NOT NULL
		)
	`)
	return err
}

func todosHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		rows, err := db.Query("SELECT id, content FROM todos ORDER BY id")
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		todos := []Todo{}
		for rows.Next() {
			var t Todo
			if err := rows.Scan(&t.ID, &t.Content); err != nil {
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}
			todos = append(todos, t)
		}

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

		var newTodo Todo
		err := db.QueryRow(
			"INSERT INTO todos (content) VALUES ($1) RETURNING id, content",
			content,
		).Scan(&newTodo.ID, &newTodo.Content)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(newTodo)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func main() {
	var err error
	db, err = connectWithRetry()
	if err != nil {
		log.Fatalf("could not connect to database: %v", err)
	}
	defer db.Close()

	if err := ensureSchema(db); err != nil {
		log.Fatalf("could not ensure schema: %v", err)
	}

	port := os.Getenv("PORT")

	http.HandleFunc("/todos", todosHandler)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
