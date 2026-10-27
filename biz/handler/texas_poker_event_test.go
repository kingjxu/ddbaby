package handler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/kingjxu/ddbaby/model"
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
	body := []byte(`{"packageName":"com.plusaa.amula","timestamp":1792526400000,"pid":"42","uid":"42",` +
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

func TestParseTexasPokerEventAcceptsAAPokerPackageAliases(t *testing.T) {
	for _, packageName := range []string{aapokerPackageName, aapokerAlternatePackageName} {
		body := fmt.Sprintf(`{"packageName":%q,"timestamp":1,"cmd":"STATE",`+
			`"direction":"ServerToClient","location":"TABLE","data":{}}`, packageName)
		if _, err := parseTexasPokerEvent([]byte(body)); err != nil {
			t.Fatalf("package %q rejected: %v", packageName, err)
		}
	}
}

func TestTexasPokerEventHTTPResponse(t *testing.T) {
	validBody := `{"packageName":"com.plusaa.amula","timestamp":1792526400000,"cmd":"STATE",` +
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

func TestTexasPokerEventLimitsHighFrequencyLargePayloadAndLogsAcceptedBodies(t *testing.T) {
	data := strings.Repeat("a", 200*1024)
	body := `{"packageName":"com.plusaa.amula","timestamp":1792526400000,"cmd":"STATE",` +
		`"direction":"ServerToClient","location":"TABLE","data":"` + data + `"}`
	now := time.Unix(1792526400, 0)
	guard := newEventIngressGuard(3, 2)

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
	if count := strings.Count(logs.String(), "full_event_logged=true"); count != 2 {
		t.Fatalf("full event log count = %d, want 2", count)
	}
	if count := strings.Count(logs.String(), ` event=`); count != 2 {
		t.Fatalf("logged event body count = %d, want 2", count)
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
	guard := newEventIngressGuard(3, 2)
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
		"bad json":        `{`,
		"missing package": `{"timestamp":1,"cmd":"STATE","direction":"ServerToClient","location":"TABLE","data":{}}`,
		"unknown package": `{"packageName":"com.unknown.poker","timestamp":1,"cmd":"STATE","direction":"ServerToClient","location":"TABLE","data":{}}`,
		"missing data":    `{"packageName":"com.plusaa.amula","timestamp":1,"cmd":"STATE","direction":"ServerToClient","location":"TABLE"}`,
		"bad command":     `{"packageName":"com.plusaa.amula","timestamp":1,"cmd":"bad command","direction":"ServerToClient","location":"TABLE","data":{}}`,
		"bad direction":   `{"packageName":"com.plusaa.amula","timestamp":1,"cmd":"STATE","direction":"sideways","location":"TABLE","data":{}}`,
		"bad location":    `{"packageName":"com.plusaa.amula","timestamp":1,"cmd":"STATE","direction":"ServerToClient","location":"LOGIN","data":{}}`,
		"fractional ts":   `{"packageName":"com.plusaa.amula","timestamp":1.5,"cmd":"STATE","direction":"ServerToClient","location":"TABLE","data":{}}`,
		"exponent ts":     `{"packageName":"com.plusaa.amula","timestamp":1e3,"cmd":"STATE","direction":"ServerToClient","location":"TABLE","data":{}}`,
		"large uid":       `{"packageName":"com.plusaa.amula","timestamp":1,"uid":"` + strings.Repeat("a", 129) + `","cmd":"STATE","direction":"ServerToClient","location":"TABLE","data":{}}`,
		"boolean uid":     `{"packageName":"com.plusaa.amula","timestamp":1,"uid":true,"cmd":"STATE","direction":"ServerToClient","location":"TABLE","data":{}}`,
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
	body := []byte(`{"packageName":"com.plusaa.amula","timestamp":1792526400000,"pid":42,"uid":7,` +
		`"PacketId":"opaque","cmd":"STATE","direction":"ServerToClient",` +
		`"data":"opaque-payload","location":"TABLE","seq":"opaque","dataExtra":9}`)
	if _, err := parseTexasPokerEvent(body); err != nil {
		t.Fatalf("parseTexasPokerEvent() error = %v", err)
	}
}

func TestParseTexasPokerEventDataSizeBoundary(t *testing.T) {
	dataAtLimit := `"` + strings.Repeat("a", 480*1024-2) + `"`
	bodyAtLimit := `{"packageName":"com.plusaa.amula","timestamp":1,"cmd":"STATE","direction":"ServerToClient",` +
		`"location":"TABLE","data":` + dataAtLimit + `}`
	if _, err := parseTexasPokerEvent([]byte(bodyAtLimit)); err != nil {
		t.Fatalf("480 KiB data rejected: %v", err)
	}

	dataOverLimit := `"` + strings.Repeat("a", 480*1024-1) + `"`
	bodyOverLimit := `{"packageName":"com.plusaa.amula","timestamp":1,"cmd":"STATE","direction":"ServerToClient",` +
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

func TestTexasPokerEventRoutesDpzxToFullLogOnlyAndRejectsUnassignedSources(t *testing.T) {
	previousRegistry := texasPokerEventStates
	previousExecute := executeAAPokerDecisionForEvent
	texasPokerEventStates = newAAPokerEventRegistry()
	decisionCalls := 0
	executeAAPokerDecisionForEvent = func(
		_ context.Context, _, _ string, _, _ int32, _ *model.TexasGtoDecisionReq,
	) (*aapokerDecisionResult, error) {
		decisionCalls++
		return nil, errors.New("unexpected decision")
	}
	t.Cleanup(func() {
		texasPokerEventStates = previousRegistry
		executeAAPokerDecisionForEvent = previousExecute
	})

	var logs bytes.Buffer
	previousOutput := logrus.StandardLogger().Out
	logrus.SetOutput(&logs)
	t.Cleanup(func() { logrus.SetOutput(previousOutput) })

	guard := newEventIngressGuard(10, 10)
	now := time.Unix(1792526400, 0)
	dpzxBody := `{"packageName":"com.dpzx.pro","timestamp":1792526400000,` +
		`"cmd":"401","direction":"ServerToClient","location":"TABLE",` +
		`"data":{"marker":"dpzx-private-hand"}}`
	var dpzxContext app.RequestContext
	dpzxContext.Request.SetBodyString(dpzxBody)
	texasPokerEventWithIngress(context.Background(), &dpzxContext, guard, now)
	if dpzxContext.Response.StatusCode() != consts.StatusOK ||
		strings.Contains(string(dpzxContext.Response.Body()), `"Result"`) {
		t.Fatalf("DPZX response = %d %s", dpzxContext.Response.StatusCode(), dpzxContext.Response.Body())
	}
	if decisionCalls != 0 || len(texasPokerEventStates.sessions) != 0 {
		t.Fatalf("DPZX entered AAPoker decision path: calls=%d sessions=%d",
			decisionCalls, len(texasPokerEventStates.sessions))
	}
	if !strings.Contains(logs.String(), "package_name=com.dpzx.pro") ||
		!strings.Contains(logs.String(), "dpzx-private-hand") ||
		!strings.Contains(logs.String(), "full_event_logged=true") {
		t.Fatalf("DPZX full event was not logged: %s", logs.String())
	}

	for name, body := range map[string]string{
		"missing": `{"timestamp":1792526400001,"cmd":"401","direction":"ServerToClient","location":"TABLE","data":{"marker":"missing-source-secret"}}`,
		"unknown": `{"packageName":"com.unknown.poker","timestamp":1792526400002,"cmd":"401","direction":"ServerToClient","location":"TABLE","data":{"marker":"unknown-source-secret"}}`,
	} {
		t.Run(name, func(t *testing.T) {
			logs.Reset()
			var requestContext app.RequestContext
			requestContext.Request.SetBodyString(body)
			texasPokerEventWithIngress(context.Background(), &requestContext, guard, now)
			if requestContext.Response.StatusCode() != consts.StatusBadRequest {
				t.Fatalf("status = %d, want 400", requestContext.Response.StatusCode())
			}
			response := string(requestContext.Response.Body())
			if strings.Contains(response, "secret") || strings.Contains(response, body) {
				t.Fatalf("400 response leaked request body: %s", response)
			}
			if strings.Contains(logs.String(), "secret") || strings.Contains(logs.String(), " event=") {
				t.Fatalf("rejected body leaked to full log: %s", logs.String())
			}
		})
	}
}

func TestTexasPokerEventRetriesFailureAndReplaysSuccessfulTurn(t *testing.T) {
	previousRegistry := texasPokerEventStates
	previousExecute := executeAAPokerDecisionForEvent
	texasPokerEventStates = newAAPokerEventRegistry()
	calls := 0
	executeAAPokerDecisionForEvent = func(
		_ context.Context, _, _ string, _, _ int32, _ *model.TexasGtoDecisionReq,
	) (*aapokerDecisionResult, error) {
		calls++
		if calls == 1 {
			return nil, fmt.Errorf("temporary auth failure")
		}
		return &aapokerDecisionResult{
			Result: "Call 8", HeroCard: []string{"A-D", "K-C"},
			DecisionAction: "call", DecisionAmount: 8, HeroMaxTarget: 8,
		}, nil
	}
	t.Cleanup(func() {
		texasPokerEventStates = previousRegistry
		executeAAPokerDecisionForEvent = previousExecute
	})

	guard := newEventIngressGuard(100, 100)
	now := time.Unix(1792526400, 0)
	events := []string{
		`{"mySeatID":0,"playerIDArray":[1762080,2836313,2646258,2838496,2859796,-1,2860755,1081977],"operationID":4,"operationRound":16}`,
		`{"bankerID":0,"operationID":4,"firstCardArray":[12,-1,-1,-1,-1,-1,-1,-1],"SecondCardArray":[24,-1,-1,-1,-1,-1,-1,-1],"smallChip":2,"bigChip":4,"chipsArray":[762,996,392,1557,606,-1,768,1020],"canPlayStatus":[1,1,1,1,1,0,1,1],"seatStatus":[2,2,2,2,2,0,2,2],"handNum":3,"straddlePop":[0,0,0,8,0,0,0,0],"operationRound":17,"mushroomPool":40,"pool_chips":42,"anteCountArray":[4,6,8,12,4,0,4,4]}`,
		`{"seatID":4,"action":6,"operationID":6,"anteNumber":0,"operationRound":18}`,
		`{"seatID":6,"action":2,"operationID":7,"anteNumber":8,"operationRound":19}`,
		`{"seatID":7,"action":2,"operationID":0,"anteNumber":8,"operationRound":20}`,
	}
	commands := []string{"REQ_GAME_ENTER_ROOM", "REQ_GAME_RECV_START_INFOR", "REQ_GAME_RECV_ACTION", "REQ_GAME_RECV_ACTION", "REQ_GAME_RECV_ACTION"}
	for i := 0; i < len(events)-1; i++ {
		ctx := postAAPokerTestEvent(t, guard, now, commands[i], events[i])
		if ctx.Response.StatusCode() != consts.StatusOK {
			t.Fatalf("setup event %d status = %d", i, ctx.Response.StatusCode())
		}
		now = now.Add(time.Millisecond)
	}

	failed := postAAPokerTestEvent(t, guard, now, commands[4], events[4])
	if failed.Response.StatusCode() != consts.StatusServiceUnavailable || calls != 1 {
		t.Fatalf("temporary failure status/calls = %d/%d", failed.Response.StatusCode(), calls)
	}
	now = now.Add(time.Millisecond)
	succeeded := postAAPokerTestEvent(t, guard, now, commands[4], events[4])
	if succeeded.Response.StatusCode() != consts.StatusOK ||
		!strings.Contains(string(succeeded.Response.Body()), `"Result":"Call 8"`) || calls != 2 {
		t.Fatalf("retry response/calls = %s/%d", succeeded.Response.Body(), calls)
	}
	now = now.Add(time.Millisecond)
	replayed := postAAPokerTestEvent(t, guard, now, commands[4], events[4])
	if replayed.Response.StatusCode() != consts.StatusOK ||
		!strings.Contains(string(replayed.Response.Body()), `"Result":"Call 8"`) || calls != 2 {
		t.Fatalf("cached replay/calls = %s/%d", replayed.Response.Body(), calls)
	}
}

func postAAPokerTestEvent(
	t *testing.T, guard *eventIngressGuard, now time.Time, command, data string,
) *app.RequestContext {
	t.Helper()
	body := fmt.Sprintf(`{"packageName":"com.plusaa.amula","uuid":"user@example.com","vpip":0,"pfr":0,"timestamp":%d,"uid":1762080,"cmd":%q,"direction":"ServerToClient","location":"TABLE","dataExtra":99,"data":%s}`,
		now.UnixMilli(), command, data)
	requestContext := &app.RequestContext{}
	requestContext.Request.SetBodyString(body)
	requestContext.Request.Header.Set("Authorization", "Bearer session-token")
	texasPokerEventWithIngress(context.Background(), requestContext, guard, now)
	return requestContext
}
