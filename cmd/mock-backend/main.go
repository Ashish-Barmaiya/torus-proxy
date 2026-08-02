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

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		fmt.Fprintf(
			w,
			`{"status":"ok","backend":"localhost:%d"}`,
			port,
		)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Optional delay ?sleep= duration to simulate slow responses
		sleep := 0 * time.Second
		if s := r.URL.Query().Get("sleep"); s != "" {
			d, err := time.ParseDuration(s)
			if err == nil {
				sleep = d
			}
		}

		time.Sleep(sleep)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		fmt.Fprintf(
			w,
			`{"status":"ok","backend":"localhost:%d"}`,
			port,
		)
	})

	addr := fmt.Sprintf(":%d", port)

	fmt.Printf("Mock backend listening on :%s\n", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		fmt.Printf("Error starting server: %v\n", err)
		os.Exit(1)
	}
}
