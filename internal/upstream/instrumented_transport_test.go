package upstream

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"torus-proxy/internal/observability"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestInstrumentedTransportSkipsMetricsWhenDisabled(t *testing.T) {
	observability.ResetForTesting()

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	base := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("ok")),
			Header:     make(http.Header),
		}, nil
	})

	tr := newInstrumentedTransport(base, false, "backend-a")

	if _, err := tr.RoundTrip(req); err != nil {
		t.Fatalf("round trip failed: %v", err)
	}

	recorder := httptest.NewRecorder()
	observability.Handler().ServeHTTP(recorder, req)

	body := recorder.Body.String()
	if strings.Contains(body, `torus_backend_requests_total{backend="backend-a"}`) {
		t.Fatalf("expected disabled transport to skip backend request metrics, got body:\n%s", body)
	}
}
