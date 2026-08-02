package upstream

import (
	"net/http"
	"time"
	"torus-proxy/internal/observability"
)

type instrumentedTransport struct {
	base    http.RoundTripper
	enabled bool
	backend string
}

func newInstrumentedTransport(base http.RoundTripper, enabled bool, backend string) http.RoundTripper {
	return &instrumentedTransport{
		base:    base,
		enabled: enabled,
		backend: backend,
	}
}

func (t *instrumentedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()

	resp, err := t.base.RoundTrip(req)

	if t.enabled {
		observability.RecordBackendRequest(t.backend)
		observability.ObserveBackendRequestDuration(
			t.backend,
			time.Since(start),
		)
	}

	if err != nil && t.enabled {
		observability.RecordBackendError(t.backend)
	}

	return resp, err
}
