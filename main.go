package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Note struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

func main() {
	port := env("PORT", "8080")

	// get database url from the environment
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is required")
	}
	
	//make a pgx connection pool
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	// wait briefly for Postgres DB to be ready by pinging it and retrying if needed
	for i := 0; i < 25; i++ {
		if err := pool.Ping(ctx); err == nil {
			break
		}
		time.Sleep(400 * time.Millisecond)
	}

	// create notes table if not exists with SQL
	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS notes (
			id SERIAL PRIMARY KEY,
			title TEXT NOT NULL,
			body  TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
	`)
	if err != nil {
		log.Fatalf("create table: %v", err)
	}

	mux := http.NewServeMux()

	//health check endpoint to check if server is running
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// POST /notes  (create)
	// GET  /notes  (list)
	mux.HandleFunc("/notes", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			createNote(w, r, pool)
		case http.MethodGet:
			listNotes(w, r, pool)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// DELETE /notes/{id}
	mux.HandleFunc("/notes/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		//parse the id from the url
		idStr := strings.TrimPrefix(r.URL.Path, "/notes/")
		id, err := strconv.Atoi(idStr)
		if err != nil || id <= 0 {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}

		//delete the note from the DB
		deleteNote(w, r, pool, id)
	})

	log.Printf("listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func createNote(w http.ResponseWriter, r *http.Request, pool *pgxpool.Pool) {\
	//parse input JSON
	var in struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	in.Title = strings.TrimSpace(in.Title)
	in.Body = strings.TrimSpace(in.Body)
	if in.Title == "" || in.Body == "" {
		http.Error(w, "title and body required", http.StatusBadRequest)
		return
	}
	//timeout context for db operation
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	var n Note

	//insert row into DB and return the created note
	err := pool.QueryRow(ctx,
		`INSERT INTO notes(title, body) VALUES ($1, $2) RETURNING id, title, body, created_at`,
		in.Title, in.Body,
	).Scan(&n.ID, &n.Title, &n.Body, &n.CreatedAt)
	if err != nil {
		http.Error(w, "db insert failed", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, n)
}

func listNotes(w http.ResponseWriter, r *http.Request, pool *pgxpool.Pool) {
	//timeout context for db operation
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	//query all notes from DB
	rows, err := pool.Query(ctx, `SELECT id, title, body, created_at FROM notes ORDER BY id DESC`)
	if err != nil {
		http.Error(w, "db query failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	notes := []Note{}
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.ID, &n.Title, &n.Body, &n.CreatedAt); err != nil {
			http.Error(w, "db scan failed", http.StatusInternalServerError)
			return
		}
		notes = append(notes, n)
	}

	writeJSON(w, http.StatusOK, notes)
}

func deleteNote(w http.ResponseWriter, r *http.Request, pool *pgxpool.Pool, id int) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	tag, err := pool.Exec(ctx, `DELETE FROM notes WHERE id=$1`, id)
	if err != nil {
		http.Error(w, "db delete failed", http.StatusInternalServerError)
		return
	}
	if tag.RowsAffected() == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
