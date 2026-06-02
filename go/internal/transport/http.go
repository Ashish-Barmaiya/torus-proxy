package transport

import (
	"net/http"
	"net/http/httputil"
)

func Forward(w http.ResponseWriter, r *http.Request, proxy *httputil.ReverseProxy) {
	proxy.ServeHTTP(w, r)
}
