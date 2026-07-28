package middleware

import "net/http"

// statusRecorder wraps an http.ResponseWriter and records
// the final HTTP status code written to the response
type statusRecorder struct {
	http.ResponseWriter
	status int
}

// NewStatusRecorder creates a recorder with the default HTTP status
// If WriteHeader is never called, net/http implicitly writes 200 OK
func NewStatusRecorder(w http.ResponseWriter) *statusRecorder {
	return &statusRecorder{
		ResponseWriter: w,
		status:         http.StatusOK,
	}
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// Status returns the final HTTP status code
func (r *statusRecorder) Status() int {
	return r.status
}
