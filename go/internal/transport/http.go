package transport

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

func Forward(w http.ResponseWriter, r *http.Request, target string) {
	u, _ := url.Parse(target)
	proxy := httputil.NewSingleHostReverseProxy(u)
	proxy.ServeHTTP(w, r)
}

