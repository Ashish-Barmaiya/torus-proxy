package upstream

import (
	"net/http"
	"time"
	"torus-proxy/internal/observability"
)

type instrumentedTransport struct {
	base    http.RoundTripper
	backend string
}

func newInstrumentedTransport(base http.RoundTripper, backend string) http.RoundTripper {
	return &instrumentedTransport{
		base:    base,
		backend: backend,
	}
}

func (t *instrumentedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	observability.RecordBackendRequest(t.backend)
	start := time.Now()

	resp, err := t.base.RoundTrip(req)

	observability.ObserveBackendRequestDuration(
		t.backend,
		time.Since(start),
	)

	if err != nil {
		observability.RecordBackendError(t.backend)
	}

	return resp, err
}
