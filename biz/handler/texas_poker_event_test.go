package handler

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

type countingReader struct {
	reader io.Reader
	read   int
}

func (r *countingReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	r.read += n
	return n, err
}

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

func TestReadTexasPokerEventBodyLimitsStreamBeforeFullBuffer(t *testing.T) {
	source := &countingReader{reader: bytes.NewReader(
		bytes.Repeat([]byte("x"), maxTexasPokerEventBytes+64*1024),
	)}
	var requestContext app.RequestContext
	requestContext.Request.SetBodyStream(source, -1)

	if _, err := readTexasPokerEventBody(&requestContext); err == nil {
		t.Fatal("expected oversized stream to be rejected")
	}
	if source.read != maxTexasPokerEventBytes+1 {
		t.Fatalf("read %d bytes, want %d", source.read, maxTexasPokerEventBytes+1)
	}
}

func TestReadTexasPokerEventBodyRejectsOversizedContentLengthWithoutReading(t *testing.T) {
	source := &countingReader{reader: strings.NewReader("not read")}
	var requestContext app.RequestContext
	requestContext.Request.SetBodyStream(source, maxTexasPokerEventBytes+1)

	if _, err := readTexasPokerEventBody(&requestContext); err == nil {
		t.Fatal("expected oversized content length to be rejected")
	}
	if source.read != 0 {
		t.Fatalf("read %d bytes, want 0", source.read)
	}
}

func TestParseTexasPokerEventRejectsInvalidInput(t *testing.T) {
	tests := map[string]string{
		"bad json":      `{`,
		"missing data":  `{"timestamp":1,"cmd":"STATE","direction":"ServerToClient","location":"TABLE"}`,
		"bad command":   `{"timestamp":1,"cmd":"bad command","direction":"ServerToClient","location":"TABLE","data":{}}`,
		"bad direction": `{"timestamp":1,"cmd":"STATE","direction":"sideways","location":"TABLE","data":{}}`,
		"bad location":  `{"timestamp":1,"cmd":"STATE","direction":"ServerToClient","location":"LOGIN","data":{}}`,
		"fractional ts": `{"timestamp":1.5,"cmd":"STATE","direction":"ServerToClient","location":"TABLE","data":{}}`,
		"exponent ts":   `{"timestamp":1e3,"cmd":"STATE","direction":"ServerToClient","location":"TABLE","data":{}}`,
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

func TestParseTexasPokerEventDataSizeBoundary(t *testing.T) {
	dataAtLimit := `"` + strings.Repeat("a", 480*1024-2) + `"`
	bodyAtLimit := `{"timestamp":1,"cmd":"STATE","direction":"ServerToClient",` +
		`"location":"TABLE","data":` + dataAtLimit + `}`
	if _, err := parseTexasPokerEvent([]byte(bodyAtLimit)); err != nil {
		t.Fatalf("480 KiB data rejected: %v", err)
	}

	dataOverLimit := `"` + strings.Repeat("a", 480*1024-1) + `"`
	bodyOverLimit := `{"timestamp":1,"cmd":"STATE","direction":"ServerToClient",` +
		`"location":"TABLE","data":` + dataOverLimit + `}`
	if _, err := parseTexasPokerEvent([]byte(bodyOverLimit)); err == nil {
		t.Fatal("expected data over 480 KiB to be rejected")
	}
}
