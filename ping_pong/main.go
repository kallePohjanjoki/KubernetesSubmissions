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

func main() {
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"),
	)

	var err error
	for i := 0; i < 20; i++ {
		db, err = sql.Open("postgres", connStr)
		if err == nil && db.Ping() == nil {
			break
		}
		log.Println("waiting for database...")
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		log.Fatal(err)
	}

	db.Exec(`CREATE TABLE IF NOT EXISTS counter (id INT PRIMARY KEY, count INT NOT NULL)`)
	db.Exec(`INSERT INTO counter (id, count) VALUES (1, 0) ON CONFLICT (id) DO NOTHING`)

	http.HandleFunc("/pingpong", func(w http.ResponseWriter, r *http.Request) {
		var previous int
		db.QueryRow(`UPDATE counter SET count = count + 1 WHERE id = 1 RETURNING count - 1`).Scan(&previous)
		fmt.Fprintf(w, "pong %d\n", previous)
	})

	http.HandleFunc("/pings", func(w http.ResponseWriter, r *http.Request) {
		var current int
		db.QueryRow("SELECT count FROM counter WHERE id = 1").Scan(&current)
		fmt.Fprint(w, strconv.Itoa(current))
	})

	port := os.Getenv("PORT")
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
