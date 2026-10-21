package handler

import (
	"context"
	"strings"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

func TestParseTexasPokerEvent(t *testing.T) {
	body := []byte(`{"timestamp":1792526400000,"pid":"42","uid":"42",` +
		`"PacketId":123,"cmd":"RSP_GAME_STATE","direction":"ServerToClient",` +
		`"data":{"table":7},"location":"TABLE","seq":1,"dataExtra":"7"}`)
	event, err := parseTexasPokerEvent(body)
	if err != nil {
		t.Fatalf("parseTexasPokerEvent() error = %v", err)
	}
	if event.Cmd != "RSP_GAME_STATE" || event.Direction != "ServerToClient" || event.Location != "TABLE" {
		t.Fatalf("unexpected event: %+v", event)
	}
}

func TestTexasPokerEventHTTPResponse(t *testing.T) {
	validBody := `{"timestamp":1792526400000,"cmd":"STATE",` +
		`"direction":"ServerToClient","location":"TABLE","data":{}}`
	var validContext app.RequestContext
	validContext.Request.SetBodyString(validBody)
	TexasPokerEvent(context.Background(), &validContext)
	if validContext.Response.StatusCode() != consts.StatusOK {
		t.Fatalf("valid status = %d, want 200", validContext.Response.StatusCode())
	}
	if !strings.Contains(string(validContext.Response.Body()), `"StatusCode":0`) {
		t.Fatalf("valid response = %s", validContext.Response.Body())
	}

	var invalidContext app.RequestContext
	invalidContext.Request.SetBodyString(`{"timestamp":1}`)
	TexasPokerEvent(context.Background(), &invalidContext)
	if invalidContext.Response.StatusCode() != consts.StatusBadRequest {
		t.Fatalf("invalid status = %d, want 400", invalidContext.Response.StatusCode())
	}
}

func TestParseTexasPokerEventRejectsInvalidInput(t *testing.T) {
	tests := map[string]string{
		"bad json":      `{`,
		"missing data":  `{"timestamp":1,"cmd":"STATE","direction":"ServerToClient","location":"TABLE"}`,
		"bad command":   `{"timestamp":1,"cmd":"bad command","direction":"ServerToClient","location":"TABLE","data":{}}`,
		"bad direction": `{"timestamp":1,"cmd":"STATE","direction":"sideways","location":"TABLE","data":{}}`,
		"bad location":  `{"timestamp":1,"cmd":"STATE","direction":"ServerToClient","location":"LOGIN","data":{}}`,
		"large uid":     `{"timestamp":1,"uid":"` + strings.Repeat("a", 129) + `","cmd":"STATE","direction":"ServerToClient","location":"TABLE","data":{}}`,
		"boolean uid":   `{"timestamp":1,"uid":true,"cmd":"STATE","direction":"ServerToClient","location":"TABLE","data":{}}`,
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := parseTexasPokerEvent([]byte(body)); err == nil {
				t.Fatal("expected invalid event to be rejected")
			}
		})
	}
}

func TestParseTexasPokerEventAcceptsNumericMetadata(t *testing.T) {
	body := []byte(`{"timestamp":1792526400000,"pid":42,"uid":7,` +
		`"PacketId":"opaque","cmd":"STATE","direction":"ServerToClient",` +
		`"data":"opaque-payload","location":"TABLE","seq":"opaque","dataExtra":9}`)
	if _, err := parseTexasPokerEvent(body); err != nil {
		t.Fatalf("parseTexasPokerEvent() error = %v", err)
	}
}
