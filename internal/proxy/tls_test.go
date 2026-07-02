package proxy

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"torus-proxy/internal/routing"
	"torus-proxy/internal/service"
	"torus-proxy/internal/upstream"
)

func TestTLSIntegration(t *testing.T) {
	// Create a HTTP backend
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proto := r.Header.Get("X-Forwarded-Proto")
		_, _ = w.Write([]byte(proto))
	}))
	defer backend.Close()

	// Generate an in-memory TLS config
	tlsCfg, err := generateTestTLSConfig()
	if err != nil {
		t.Fatal(err)
	}

	b, err := upstream.NewBackend(backend.URL)
	if err != nil {
		t.Fatal(err)
	}
	b.Proxy.Transport.(*http.Transport).ResponseHeaderTimeout = 10 * time.Second

	svc := service.NewService([]*upstream.Backend{b})
	router := routing.NewRouter()
	router.AddRoute("/api", svc)

	srv := NewServer(router, testLogger, tlsCfg)

	// Start torus server on a random port with TLS
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()

	mux := http.NewServeMux()
	mux.Handle("/", srv.Handler())

	baseCtx, forceCancel := context.WithCancel(context.Background())
	srv.baseCtx = baseCtx
	srv.forceCancel = forceCancel

	srv.srv = &http.Server{
		Addr:         addr,
		Handler:      mux,
		BaseContext:  func(l net.Listener) context.Context { return baseCtx },
		TLSConfig:    tlsCfg,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	srv.ready.Store(true)

	go func() {
		_ = srv.srv.ServeTLS(listener, "", "")
	}()

	// Wait for server to be ready and then send HTTPS request
	time.Sleep(200 * time.Millisecond)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	resp, err := client.Get(fmt.Sprintf("https://%s/api", addr))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "https" {
		t.Errorf("expected X-Forwarded-Proto 'https', got '%s'", string(body))
	}

	if err := srv.Shutdown(5 * time.Second); err != nil {
		t.Errorf("shutdown error: %v", err)
	}
}

// generateTestTLSConfig creates an in-memory TLS config with a self-signed cert.
func generateTestTLSConfig() (*tls.Config, error) {
	// Generate a private key
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generating key: %w", err)
	}

	// Generate a serial number
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("generating serial: %w", err)
	}

	// Certificate template
	template := x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			Organization: []string{"Torus Test"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		DNSNames:              []string{"localhost"},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	// Create certificate
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return nil, fmt.Errorf("creating certificate: %w", err)
	}

	// Encode certificate to PEM
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	// Marshal private key
	keyBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return nil, fmt.Errorf("marshaling private key: %w", err)
	}

	// Encode key to PEM
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})

	// Build TLS certificate
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, fmt.Errorf("building key pair: %w", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}, nil
}
