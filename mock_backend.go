// mock_backend.go
package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"
)

func main() {
	// Default port 3001, override with first command-line argument
	port := 3001
	if len(os.Args) > 1 {
		p, err := strconv.Atoi(os.Args[1])
		if err == nil {
			port = p
		}
	}

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Accept an optional ?sleep= duration to simulate slow responses
		sleep := 0 * time.Second
		if s := r.URL.Query().Get("sleep"); s != "" {
			d, err := time.ParseDuration(s)
			if err == nil {
				sleep = d
			}
		}

		// fmt.Printf("Backend :%d received request, sleeping %v...\n", port, sleep)
		time.Sleep(sleep)
		// fmt.Printf("Backend :%d responding OK\n", port)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	fmt.Printf("Mock backend listening on :%d\n", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
		fmt.Printf("Error starting server: %v\n", err)
	}
}
