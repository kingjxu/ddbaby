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
