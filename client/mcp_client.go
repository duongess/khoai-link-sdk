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

// MCPClient chua URL va HTTP Transport dung chung
type MCPClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewMCPClient(baseURL string, timeout time.Duration) *MCPClient {
	return &MCPClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: timeout},
	}
}

// MCPRegister dang ky config cua Node len MCP Gateway
func (c *MCPClient) MCPRegister(ctx context.Context, payload any) (*core.Response, error) {
	targetURL, err := url.JoinPath(c.baseURL, "/api/v1/nodes/register")
	if err != nil {
		return nil, fmt.Errorf("invalid register url: %w", err)
	}

	return c.doJSONRequest(ctx, http.MethodPost, targetURL, payload)
}

// MCPHeartbeat gui tin hieu song dinh ky
func (c *MCPClient) MCPHeartbeat(ctx context.Context, nodeID string) (*core.Response, error) {
	targetURL, err := url.JoinPath(c.baseURL, "/api/v1/nodes/heartbeat")
	if err != nil {
		return nil, fmt.Errorf("invalid heartbeat url: %w", err)
	}

	payload := map[string]string{"node_id": nodeID}
	return c.doJSONRequest(ctx, http.MethodPost, targetURL, payload)
}

// MCPFetchPeers lay danh ba IP cua cac node khac
func (c *MCPClient) MCPFetchPeers(ctx context.Context) (*core.Response, error) {
	targetURL, err := url.JoinPath(c.baseURL, "/api/v1/peers")
	if err != nil {
		return nil, fmt.Errorf("invalid fetch peers url: %w", err)
	}

	return c.doJSONRequest(ctx, http.MethodGet, targetURL, nil)
}

func (c *MCPClient) doJSONRequest(ctx context.Context, method string, targetURL string, payload any) (*core.Response, error) {
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
