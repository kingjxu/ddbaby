package gto

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServerAuthAndDecisionContract(t *testing.T) {
	logger := log.New(io.Discard, "", 0)
	service, err := New(Options{TablesDir: "testdata", Samples: 300, Logger: logger})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	server := httptest.NewServer(NewServer(service, "secret", logger).Handler())
	defer server.Close()

	req := preflopRequest("BTN", []string{"A-S", "A-H"})
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}

	if status := postStatus(t, server.URL+"/gto/decision", body, ""); status != http.StatusUnauthorized {
		t.Fatalf("missing key status = %d, want 401", status)
	}
	if status := postStatus(t, server.URL+"/gto/decision", body, "wrong"); status != http.StatusUnauthorized {
		t.Fatalf("wrong key status = %d, want 401", status)
	}

	for _, path := range []string{"/gto/decision", "/gto/decision_v2"} {
		status, payload := postJSON(t, server.URL+path, body, "secret")
		if status != http.StatusOK {
			t.Fatalf("%s status = %d, want 200: %s", path, status, payload)
		}
		var resp TexasGtoDecisionResp
		if err := json.Unmarshal(payload, &resp); err != nil {
			t.Fatalf("%s decode: %v", path, err)
		}
		if resp.Action != "raise" || resp.RaiseSize != 250 {
			t.Fatalf("%s action/size = %s/%d, want raise/250", path, resp.Action, resp.RaiseSize)
		}
	}
}

func TestServerRejectsInvalidStage(t *testing.T) {
	logger := log.New(io.Discard, "", 0)
	service, err := New(Options{TablesDir: "testdata", Samples: 300, Logger: logger})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	server := httptest.NewServer(NewServer(service, "", logger).Handler())
	defer server.Close()

	req := preflopRequest("BTN", []string{"A-S", "A-H"})
	req.GameStage = "showdown"
	body, _ := json.Marshal(req)
	if status := postStatus(t, server.URL+"/gto/decision", body, ""); status != http.StatusBadRequest {
		t.Fatalf("invalid stage status = %d, want 400", status)
	}
}

func TestServerHealth(t *testing.T) {
	logger := log.New(io.Discard, "", 0)
	service, err := New(Options{TablesDir: "testdata", Samples: 300, Logger: logger})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	server := httptest.NewServer(NewServer(service, "secret", logger).Handler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("health status = %d, want 200", resp.StatusCode)
	}
	var payload map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload["status"] != "ok" {
		t.Fatalf("health payload = %v", payload)
	}
}

func postStatus(t *testing.T, url string, body []byte, key string) int {
	t.Helper()
	status, _ := postJSON(t, url, body, key)
	return status
}

func postJSON(t *testing.T, url string, body []byte, key string) (int, []byte) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set("X-API-Key", key)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, data
}
