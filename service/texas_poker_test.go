package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/kingjxu/ddbaby/model"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestRecognizePokerByFilePathKeepsHeroTurnGate(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "poker-*.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = file.Write([]byte("image-content")); err != nil {
		t.Fatal(err)
	}
	if err = file.Close(); err != nil {
		t.Fatal(err)
	}

	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })

	for _, isHeroTurn := range []bool{false, true} {
		t.Run(map[bool]string{false: "non hero", true: "hero"}[isHeroTurn], func(t *testing.T) {
			var sent model.RecognizeRequest
			http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if err := json.NewDecoder(req.Body).Decode(&sent); err != nil {
					t.Fatal(err)
				}
				body := `{"hero_info":{"is_hero_turn":` + map[bool]string{false: "false", true: "true"}[isHeroTurn] + `}}`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(body)),
					Header:     make(http.Header),
				}, nil
			})

			result, err := RecognizePokerByFilePath(context.Background(), file.Name(), false)
			if err != nil {
				t.Fatal(err)
			}
			if sent.ParseAll {
				t.Fatal("file path recognition sent parse_all=true, want Hero-turn gate enabled")
			}
			if result.HeroInfo.IsHeroTurn != isHeroTurn {
				t.Fatalf("is_hero_turn = %v, want %v", result.HeroInfo.IsHeroTurn, isHeroTurn)
			}
		})
	}
}
