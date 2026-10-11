package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/kingjxu/ddbaby/model"
	"github.com/stretchr/testify/require"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestRecognizePokerUsesLocalDezhou(t *testing.T) {
	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		require.Equal(t, "http://127.0.0.1:5555/recognize", req.URL.String())
		require.Equal(t, http.MethodPost, req.Method)

		var payload model.RecognizeRequest
		require.NoError(t, json.NewDecoder(req.Body).Decode(&payload))
		require.Equal(t, "image-data", payload.ImageBase64)
		require.Equal(t, "polerv2", payload.App)
		require.True(t, payload.ParseAll)

		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"app":"polerv2"}`)),
		}, nil
	})

	result, err := RecognizePoker(context.Background(), "image-data", true)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "polerv2", result.App)
}

func TestGtoDecisionByProfileRoutesAndMapsUpstreamFields(t *testing.T) {
	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })
	t.Setenv(gtoAPIKeyEnv, "test-api-key")

	tests := []struct {
		name     string
		vpip     int32
		pfr      int32
		wantPath string
		wantVVIP bool
	}{
		{name: "default uses original endpoint", wantPath: "/gto/decision"},
		{name: "profile uses v2 endpoint", vpip: 42, pfr: 20,
			wantPath: "/gto/decision_v2", wantVVIP: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
				require.Equal(t, tt.wantPath, req.URL.Path)
				require.Equal(t, "test-api-key", req.Header.Get("X-API-Key"))
				var payload map[string]any
				require.NoError(t, json.NewDecoder(req.Body).Decode(&payload))
				if tt.wantVVIP {
					require.Equal(t, float64(tt.vpip), payload["vvip"])
					require.Equal(t, float64(tt.pfr), payload["pfr"])
				} else {
					require.NotContains(t, payload, "vvip")
					require.NotContains(t, payload, "pfr")
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body: io.NopCloser(strings.NewReader(
						`{"action":"fold","raise_size":0}`)),
				}, nil
			})

			response, err := GtoDecisionByProfile(context.Background(),
				&model.TexasGtoDecisionReq{GameType: "cash"}, tt.vpip, tt.pfr)
			require.NoError(t, err)
			require.Equal(t, "fold", response.Action)
		})
	}
}

func TestGtoDecisionRequiresConfiguredAPIKey(t *testing.T) {
	t.Setenv(gtoAPIKeyEnv, "")
	_, err := GtoDecision(context.Background(), &model.TexasGtoDecisionReq{})
	require.ErrorContains(t, err, gtoAPIKeyEnv)
}
