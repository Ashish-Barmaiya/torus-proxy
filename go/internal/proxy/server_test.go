package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"torus-proxy/internal/routing"
	"torus-proxy/internal/service"
	"torus-proxy/internal/upstream"
)

func TestProxyFlow(t *testing.T) {
	// 1. Setup "Fake" Backend Server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("backend_response"))
	}))
	defer backend.Close()

	// 2. Setup Proxy Components
	upstreams := []*upstream.Backend{{URL: backend.URL}}
	svc := service.NewService(upstreams)
	router := routing.NewRouter()
	router.AddRoute("/api", svc)
	proxyServer := NewServer(router)

	// 3. Define the Test Table
	tests := []struct {
		name           string
		requestPath    string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Valid Route",
			requestPath:    "/api",
			expectedStatus: http.StatusOK,
			expectedBody:   "backend_response",
		},
		{
			name:           "Invalid Route",
			requestPath:    "/unknown",
			expectedStatus: http.StatusNotFound,
			expectedBody:   "Not Found\n",
		},
	}

	// 4. Run Tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fake request to our proxy
			req := httptest.NewRequest("GET", tt.requestPath, nil)
			// Create a response recorder (acts like a client)
			w := httptest.NewRecorder()

			// Directly call the httpHandler
			proxyServer.httpHandler(w, req)

			resp := w.Result()
			body, _ := io.ReadAll(resp.Body)

			// Assertions
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
			if string(body) != tt.expectedBody {
				t.Errorf("Expected body %q, got %q", tt.expectedBody, string(body))
			}
		})
	}
}