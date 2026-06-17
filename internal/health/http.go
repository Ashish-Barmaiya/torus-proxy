package health

import (
	"context"
	"fmt"
	"net/http"
)

// HTTPChecker performs a GET to health endpoint.
type HTTPChecker struct {
	URL    string
	Client *http.Client
	Path   string
}

func (h *HTTPChecker) Check(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.URL+h.Path, nil)
	if err != nil {
		return err
	}
	resp, err := h.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned %d", resp.StatusCode)
	}
	return nil
}

func (h *HTTPChecker) Target() string {
	return h.URL
}
