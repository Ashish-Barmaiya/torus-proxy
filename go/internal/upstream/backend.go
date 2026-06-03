package upstream

import (
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

type Backend struct {
	URL   string
	Proxy *httputil.ReverseProxy
}

// This creates new backend
func NewBackend(targetUrl string) (*Backend, error) {
	u, err := url.Parse(targetUrl)
	if err != nil {
		return nil, err
	}

	proxy := httputil.NewSingleHostReverseProxy(u)

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

	proxy.Transport = customTransport

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
	}

	return &Backend{
		URL:   targetUrl,
		Proxy: proxy,
	}, nil
}
