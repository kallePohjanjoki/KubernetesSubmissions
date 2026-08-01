package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/lib/pq"
)

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
		CREATE TABLE IF NOT EXISTS counter (
			id INT PRIMARY KEY,
			count INT NOT NULL
		)
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		INSERT INTO counter (id, count)
		VALUES (1, 0)
		ON CONFLICT (id) DO NOTHING
	`)
	return err
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
	if port == "" {
		port = "3000"
	}

	// The app now owns "/" directly - the Gateway rewrites
	// /pingpong -> / before the request ever reaches this pod.
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var previous int
		err := db.QueryRow(`
			UPDATE counter SET count = count + 1
			WHERE id = 1
			RETURNING count - 1
		`).Scan(&previous)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "pong %d\n", previous)
	})

	// Internal-only endpoint, called directly pod-to-pod by
	// log-output. Not exposed through the Gateway at all.
	http.HandleFunc("/pings", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var current int
		err := db.QueryRow("SELECT count FROM counter WHERE id = 1").Scan(&current)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		fmt.Fprint(w, strconv.Itoa(current))
	})

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
