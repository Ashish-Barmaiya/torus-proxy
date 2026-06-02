package upstream

import (
	"net/http"
	"net/http/httputil"
	"net/url"
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

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
	}

	return &Backend{
		URL:   targetUrl,
		Proxy: proxy,
	}, nil
}
