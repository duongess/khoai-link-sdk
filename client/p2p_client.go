package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/duongess/khoai-link-protocol/core"
)

type P2pClient struct {
	httpClient *http.Client
}

func NewP2pClient(timeout time.Duration) *P2pClient {
	if timeout == 0 {
		timeout = 15 * time.Second
	}
	return &P2pClient{
		httpClient: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

// ForwardTask ban task sang node tiep theo bang cach goi doJSONRequest
func (c *P2pClient) ForwardTask(ctx context.Context, targetEndpoint, reqID string, payload any) (*core.Response, error) {
	targetURL, err := url.JoinPath(targetEndpoint, "/api/v1/execute")
	if err != nil {
		return nil, fmt.Errorf("invalid target url: %w", err)
	}

	return c.doJSONRequest(ctx, http.MethodPost, targetURL, reqID, payload)
}

// doJSONRequest xu ly toan bo logic HTTP va parse Envelope core.Response
func (c *P2pClient) doJSONRequest(ctx context.Context, method, targetURL, reqID string, payload any) (*core.Response, error) {
	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshal payload: %w", err)
		}
		body = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, targetURL, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if reqID != "" {
		req.Header.Set("X-Request-ID", reqID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request to %s failed: %w", targetURL, err)
	}
	defer resp.Body.Close()

	// Doc toi da 10MB tranh tran RAM
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return nil, fmt.Errorf("read response from %s: %w", targetURL, err)
	}

	var cr core.Response
	if err := json.Unmarshal(respBody, &cr); err != nil {
		return nil, fmt.Errorf("unmarshal response from %s (status %d): %w", targetURL, resp.StatusCode, err)
	}

	if resp.StatusCode >= 400 {
		return &cr, fmt.Errorf("peer returned error %d: %s", resp.StatusCode, cr.Message)
	}

	return &cr, nil
}
