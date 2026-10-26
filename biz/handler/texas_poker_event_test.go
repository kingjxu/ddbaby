package handler

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/sirupsen/logrus"
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

func TestTexasPokerEventLimitsHighFrequencyLargePayloadAndLogBytes(t *testing.T) {
	data := strings.Repeat("a", 200*1024)
	body := `{"timestamp":1792526400000,"cmd":"STATE",` +
		`"direction":"ServerToClient","location":"TABLE","data":"` + data + `"}`
	now := time.Unix(1792526400, 0)
	guard := newEventIngressGuard(3, 2, eventFullLogCost(len(body))+128, now.Add(time.Hour))

	var logs bytes.Buffer
	previousOutput := logrus.StandardLogger().Out
	logrus.SetOutput(&logs)
	t.Cleanup(func() { logrus.SetOutput(previousOutput) })

	statuses := make([]int, 0, 4)
	for i := 0; i < 4; i++ {
		var requestContext app.RequestContext
		requestContext.Request.SetBodyString(body)
		texasPokerEventWithIngress(context.Background(), &requestContext, guard, now)
		statuses = append(statuses, requestContext.Response.StatusCode())
	}
	wantStatuses := []int{consts.StatusOK, consts.StatusOK, consts.StatusTooManyRequests, consts.StatusTooManyRequests}
	for i, want := range wantStatuses {
		if statuses[i] != want {
			t.Fatalf("status[%d] = %d, want %d", i, statuses[i], want)
		}
	}
	if count := strings.Count(logs.String(), "full_event_logged=true"); count != 1 {
		t.Fatalf("full event log count = %d, want 1", count)
	}
	if count := strings.Count(logs.String(), "full_event_logged=false"); count != 1 {
		t.Fatalf("metadata-only event log count = %d, want 1", count)
	}

	var nextWindow app.RequestContext
	nextWindow.Request.SetBodyString(body)
	texasPokerEventWithIngress(context.Background(), &nextWindow, guard, now.Add(time.Minute))
	if !strings.Contains(logs.String(), "dropped_count=2") {
		t.Fatalf("missing aggregated drop count in logs")
	}
}

func TestEventIngressGuardAppliesGlobalAndSourceQuotas(t *testing.T) {
	now := time.Unix(1792526400, 0)
	guard := newEventIngressGuard(3, 2, 1024, time.Time{})
	if allowed, _ := guard.allowRequest(now, "198.51.100.1"); !allowed {
		t.Fatal("first source request was rejected")
	}
	if allowed, _ := guard.allowRequest(now, "198.51.100.1"); !allowed {
		t.Fatal("second source request was rejected")
	}
	if allowed, _ := guard.allowRequest(now, "198.51.100.1"); allowed {
		t.Fatal("source quota was not enforced")
	}
	if allowed, _ := guard.allowRequest(now, "198.51.100.2"); !allowed {
		t.Fatal("independent source request was rejected before global quota")
	}
	if allowed, _ := guard.allowRequest(now, "198.51.100.2"); allowed {
		t.Fatal("global quota was not enforced")
	}
}

func TestEventIngressGuardFullLogDeadline(t *testing.T) {
	now := time.Unix(1792526400, 0)
	guard := newEventIngressGuard(10, 10, 1024, now.Add(time.Minute))
	if !guard.allowFullLog(now, 100) {
		t.Fatal("full log was rejected before its deadline")
	}
	if guard.allowFullLog(now.Add(time.Minute), 100) {
		t.Fatal("full log remained enabled at its deadline")
	}
}

func TestFullEventLoggingUntilRequiresBoundedDeadline(t *testing.T) {
	now := time.Unix(1792526400, 0)
	t.Setenv("DDBABY_GTO_EVENT_FULL_LOG", "1")
	t.Setenv("DDBABY_GTO_EVENT_FULL_LOG_UNTIL", "")
	if deadline := fullEventLoggingUntil(now); !deadline.IsZero() {
		t.Fatalf("missing deadline enabled full logs until %v", deadline)
	}
	t.Setenv("DDBABY_GTO_EVENT_FULL_LOG_UNTIL", "1792616400")
	if deadline := fullEventLoggingUntil(now); !deadline.IsZero() {
		t.Fatalf("deadline beyond 24 hours enabled full logs until %v", deadline)
	}
	want := now.Add(2 * time.Hour)
	t.Setenv("DDBABY_GTO_EVENT_FULL_LOG_UNTIL", "1792533600")
	if deadline := fullEventLoggingUntil(now); !deadline.Equal(want) {
		t.Fatalf("deadline = %v, want %v", deadline, want)
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

func TestTexasPokerEventBearerToken(t *testing.T) {
	var requestContext app.RequestContext
	requestContext.Request.Header.Set("Authorization", "Bearer session-token")
	if got := texasPokerEventBearerToken(&requestContext); got != "session-token" {
		t.Fatalf("token = %q", got)
	}
	requestContext.Request.Header.Set("Authorization", "Bearer token with spaces")
	if got := texasPokerEventBearerToken(&requestContext); got != "" {
		t.Fatalf("invalid token accepted: %q", got)
	}
}
