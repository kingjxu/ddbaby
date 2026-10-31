package gto

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// SolverClient calls an optional L2 postflop solver sidecar.
//
// Protocol: POST {base}/solve with the same TexasGtoDecisionReq JSON body and
// a TexasGtoDecisionResp JSON response. The sidecar owns board/range
// construction; this service only routes heads-up postflop spots to it and
// falls back to the heuristic on any error or timeout.
type SolverClient struct {
	baseURL string
	client  *http.Client
}

func NewSolverClient(baseURL string, timeout time.Duration) *SolverClient {
	if timeout <= 0 {
		timeout = 1500 * time.Millisecond
	}
	return &SolverClient{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		client:  &http.Client{Timeout: timeout},
	}
}

func (c *SolverClient) Enabled() bool {
	return c != nil && c.baseURL != ""
}

func (c *SolverClient) Decide(ctx context.Context, req *TexasGtoDecisionReq) (*TexasGtoDecisionResp, error) {
	if !c.Enabled() {
		return nil, fmt.Errorf("solver is not configured")
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/solve", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("solver status %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	var out TexasGtoDecisionResp
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	if out.Action == "" {
		return nil, fmt.Errorf("solver returned empty action")
	}
	return &out, nil
}
