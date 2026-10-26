package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	taoyuanVerifyTokenURL = "http://114.132.228.98/api/auth/verify-token"
	maxAuthResponseBytes  = 64 * 1024
)

var taoyuanAuthClient = &http.Client{
	Timeout: 5 * time.Second,
	CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

type taoyuanVerifyTokenResponse struct {
	Code int `json:"code"`
	Data struct {
		Valid   bool `json:"valid"`
		Account struct {
			Email string `json:"email"`
		} `json:"account"`
	} `json:"data"`
}

// VerifyTaoyuanIdentity binds the event's uuid to the authenticated session.
// The current business API historically uses the login email as uuid.
func VerifyTaoyuanIdentity(ctx context.Context, token, uuid string) error {
	return verifyTaoyuanIdentity(ctx, taoyuanAuthClient, taoyuanVerifyTokenURL, token, uuid)
}

func verifyTaoyuanIdentity(ctx context.Context, client *http.Client, url, token, uuid string) error {
	token = strings.TrimSpace(token)
	uuid = strings.TrimSpace(uuid)
	if token == "" || len(token) > 4096 || uuid == "" || len(uuid) > 128 {
		return errors.New("missing or invalid session identity")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("verify session: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("verify session returned status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxAuthResponseBytes+1))
	if err != nil || len(body) > maxAuthResponseBytes {
		return errors.New("invalid session verification response")
	}
	var result taoyuanVerifyTokenResponse
	if err = json.Unmarshal(body, &result); err != nil {
		return errors.New("invalid session verification response")
	}
	if result.Code != 0 || !result.Data.Valid ||
		!strings.EqualFold(strings.TrimSpace(result.Data.Account.Email), uuid) {
		return errors.New("session identity does not match uuid")
	}
	return nil
}
