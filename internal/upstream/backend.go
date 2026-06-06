package upstream

import (
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

type Backend struct {
	URL     string
	Proxy   *httputil.ReverseProxy
	healthy atomic.Bool
}

// IsHealthy returns true if the backend is currently healthy
func (b *Backend) IsHealthy() bool {
	return b.healthy.Load()
}

// SetHealthy updates the health status
func (b *Backend) SetHealthy(val bool) {
	b.healthy.Store(val)
}

// This creates new backend
func NewBackend(targetUrl string) (*Backend, error) {
	u, err := url.Parse(targetUrl)
	if err != nil {
		return nil, err
	}

	proxy := httputil.NewSingleHostReverseProxy(u)

	proxy.Director = nil

	// Custom connection pool
	customTransport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          10000,
		MaxIdleConnsPerHost:   2000,
		IdleConnTimeout:       90 * time.Second,
		DisableKeepAlives:     false,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
	}

	proxy.Rewrite = func(pr *httputil.ProxyRequest) {
		pr.Out.URL.Scheme = u.Scheme
		pr.Out.URL.Host = u.Host

		// Extract client IP
		clientIP, _, err := net.SplitHostPort(pr.In.RemoteAddr)
		if err != nil {
			clientIP = pr.In.RemoteAddr
		}

		// X-Forwarder-For
		if prior := pr.In.Header.Get("X-Forwarded-For"); prior != "" {
			pr.Out.Header.Set("X-Forwarded-For", prior+", "+clientIP)
		} else {
			pr.Out.Header.Set("X-Forwarded-For", clientIP)
		}

		// X-Forwarder-Proto
		if pr.In.TLS != nil {
			pr.Out.Header.Set("X-Forwarder-Proto", "https")
		} else {
			pr.Out.Header.Set("X-Forwarder-Proto", "http")
		}

		// X-Forwarded-Host
		if host := pr.In.Host; host != "" {
			pr.Out.Header.Set("X-Forwarded-Host", host)
		}

		// X-Request-ID
		reqID := pr.In.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = uuid.NewString()
		}
		pr.Out.Header.Set("X-Request-ID", reqID)
	}

	proxy.Transport = customTransport

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("proxy error: %v", err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
	}

	b := &Backend{
		URL:   targetUrl,
		Proxy: proxy,
	}
	b.healthy.Store(true)
	return b, nil
}
