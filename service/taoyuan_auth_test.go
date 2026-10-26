package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVerifyTaoyuanIdentityRequiresMatchingAuthenticatedEmail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Header.Get("Authorization") != "Bearer valid-token" {
			t.Errorf("unexpected auth request: %s %q", r.Method, r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"data":{"valid":true,"account":{"email":"hero@example.com"}}}`))
	}))
	defer server.Close()

	if err := verifyTaoyuanIdentity(context.Background(), server.Client(), server.URL,
		"valid-token", "hero@example.com"); err != nil {
		t.Fatalf("matching identity rejected: %v", err)
	}
	if err := verifyTaoyuanIdentity(context.Background(), server.Client(), server.URL,
		"valid-token", "attacker@example.com"); err == nil {
		t.Fatal("mismatched uuid was accepted")
	}
}

func TestVerifyTaoyuanIdentityRejectsInvalidSession(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"data":{"valid":false}}`))
	}))
	defer server.Close()
	if err := verifyTaoyuanIdentity(context.Background(), server.Client(), server.URL,
		"expired", "hero@example.com"); err == nil {
		t.Fatal("invalid session was accepted")
	}
}
