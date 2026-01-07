package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

const notesPath = "/data/notes.txt"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Go + Docker volume demo")
	})

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ok")
	})

	// POST /save (body text) -> append to /data/notes.txt
	mux.HandleFunc("/save", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}

		// Make sure /data exists (works if container has permission)
		if err := os.MkdirAll("/data", 0755); err != nil {
			http.Error(w, "failed to create data dir", http.StatusInternalServerError)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil || len(body) == 0 {
			http.Error(w, "send text in request body", http.StatusBadRequest)
			return
		}

		f, err := os.OpenFile(notesPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			http.Error(w, "failed to open notes file", http.StatusInternalServerError)
			return
		}
		defer f.Close()

		if _, err := f.WriteString(string(body) + "\n"); err != nil {
			http.Error(w, "failed to write", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		fmt.Fprintln(w, "saved")
	})

	// GET /notes -> return file contents
	mux.HandleFunc("/notes", func(w http.ResponseWriter, r *http.Request) {
		b, err := os.ReadFile(notesPath)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Fprintln(w, "(no notes yet)")
				return
			}
			http.Error(w, "failed to read notes", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write(b)
	})

	addr := ":" + port
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
