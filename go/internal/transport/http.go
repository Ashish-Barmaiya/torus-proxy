package transport

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

func Forward(w http.ResponseWriter, r *http.Request, target string) {
	u, err := url.Parse(target)
	if err != nil {
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
	}

	proxy := httputil.NewSingleHostReverseProxy(u)

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
	}

	proxy.ServeHTTP(w, r)
}
