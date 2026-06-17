package main

import (
	"io"
	"log"
	"net/http"
)

func main() {
	mux1 := http.NewServeMux()
	mux1.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Backend 1 received request on path:", r.URL.Path)
		io.WriteString(w, "Response from Backend 1 (Port 3001)\n")
	})

	mux1.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "OK")
	})

	mux2 := http.NewServeMux()
	mux2.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Backend 2 received request on path:", r.URL.Path)
		io.WriteString(w, "Response from Backend 2 (Port 3002)\n")
	})

	mux2.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "OK")
	})

	//  This starts backend 1 asynchronously in a background green thread
	go func() {
		log.Println("Starting Mock Backend 1 on :3001...")
		if err := http.ListenAndServe(":3001", mux1); err != nil {
			log.Fatalf("Backend 1 crashed: %v", err)
		}
	}()

	// This starts backend 2 on the main blocking thread
	log.Println("Starting Mock Backend 2 on :3002...")
	if err := http.ListenAndServe(":3002", mux2); err != nil {
		log.Fatalf("Backend 2 crashed: %v", err)
	}
}
