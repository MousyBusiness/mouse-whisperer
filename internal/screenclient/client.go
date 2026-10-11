// Package screenclient is the main app's thin client for the screend
// screen-perception server. Deliberately dumb: prewarm is fire-and-forget
// and scan handlers fall back to the local OCR path on any error, so no
// health-check state machine is needed.
package screenclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/mousybusiness/mouse_whisperer/internal/screend/api"
)

type Client struct {
	baseURL string
	query   *http.Client
	prewarm *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		query:   &http.Client{Timeout: 2500 * time.Millisecond},
		// prewarm returns 202 immediately; anything slower means the server
		// is wedged and the utterance must not wait on it
		prewarm: &http.Client{Timeout: 300 * time.Millisecond},
	}
}

// Prewarm asks screend to start capturing now so the query at command time
// hits a hot cache. Failures are logged at debug and otherwise ignored.
func (c *Client) Prewarm() {
	body, _ := json.Marshal(api.PrewarmRequest{Provider: api.ProviderOCR})
	resp, err := c.prewarm.Post(c.baseURL+"/v1/prewarm", "application/json", bytes.NewReader(body))
	if err != nil {
		log.Debugf("screend prewarm: %v", err)
		return
	}
	resp.Body.Close()
}

func (c *Client) FindTargets(ctx context.Context, req api.TargetsRequest) (*api.TargetsResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/targets", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.query.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp api.ErrorResponse
		if json.NewDecoder(resp.Body).Decode(&errResp) == nil && errResp.Error != "" {
			return nil, fmt.Errorf("screend: %s", errResp.Error)
		}
		return nil, fmt.Errorf("screend: status %d", resp.StatusCode)
	}

	var out api.TargetsResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode screend response: %w", err)
	}
	return &out, nil
}
