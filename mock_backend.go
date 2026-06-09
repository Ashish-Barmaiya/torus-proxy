package main

import (
	"io"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "OK")
	})
	go http.ListenAndServe(":3001", nil)
	http.ListenAndServe(":3002", nil)
}
