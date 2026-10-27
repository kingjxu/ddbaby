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
		event, err := parseTexasPokerEvent([]byte(body))
		if err != nil {
			t.Fatalf("package %q rejected: %v", packageName, err)
		}
		if event.route != pokerEventRouteAAPoker {
			t.Fatalf("package %q route = %v", packageName, event.route)
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

func TestTexasPokerEventDoesNotRateLimitAndLogsEveryPayload(t *testing.T) {
	const requestCount = 301
	data := strings.Repeat("a", 4*1024)
	body := `{"packageName":"com.plusaa.amula","timestamp":1792526400000,"cmd":"STATE",` +
		`"direction":"ServerToClient","location":"TABLE","data":"` + data + `"}`
	now := time.Unix(1792526400, 0)

	var logs bytes.Buffer
	previousOutput := logrus.StandardLogger().Out
	logrus.SetOutput(&logs)
	t.Cleanup(func() { logrus.SetOutput(previousOutput) })

	statuses := make([]int, 0, requestCount)
	for i := 0; i < requestCount; i++ {
		var requestContext app.RequestContext
		requestContext.Request.SetBodyString(body)
		texasPokerEventAt(context.Background(), &requestContext, now)
		statuses = append(statuses, requestContext.Response.StatusCode())
	}
	for i, status := range statuses {
		if status != consts.StatusOK {
			t.Fatalf("status[%d] = %d, want %d", i, status, consts.StatusOK)
		}
	}
	if count := strings.Count(logs.String(), "full_event_logged=true"); count != requestCount {
		t.Fatalf("full event log count = %d, want %d", count, requestCount)
	}
	if count := strings.Count(logs.String(), ` event=`); count != requestCount {
		t.Fatalf("logged event body count = %d, want %d", count, requestCount)
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
		"bad json":           `{`,
		"array":              `[]`,
		"null":               `null`,
		"multiple values":    `{"packageName":"com.dpzx.pro"} {}`,
		"missing package":    `{"timestamp":1}`,
		"non-string package": `{"packageName":123}`,
		"unknown package":    `{"packageName":"com.unknown.poker"}`,
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := parseTexasPokerEvent([]byte(body)); err == nil {
				t.Fatal("expected invalid event to be rejected")
			}
		})
	}
}

func TestParseTexasPokerEventAcceptsUnvalidatedFields(t *testing.T) {
	body := []byte(`{"packageName":"com.dpzx.pro","timestamp":"unknown","pid":{},` +
		`"uid":true,"cmd":{"new":"shape"},"direction":7,"location":false}`)
	event, err := parseTexasPokerEvent(body)
	if err != nil {
		t.Fatalf("unvalidated fields rejected: %v", err)
	}
	if event.PackageName != dpzxPackageName || event.route != pokerEventRouteDPZX {
		t.Fatalf("unexpected route: package=%q route=%v", event.PackageName, event.route)
	}

	largeData := `"` + strings.Repeat("a", 490*1024) + `"`
	largeBody := `{"packageName":"com.dpzx.pro","data":` + largeData + `}`
	if len(largeBody) >= maxTexasPokerEventBytes {
		t.Fatalf("test body unexpectedly exceeds total body limit: %d", len(largeBody))
	}
	if _, err = parseTexasPokerEvent([]byte(largeBody)); err != nil {
		t.Fatalf("data above former 480 KiB sub-limit rejected: %v", err)
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

func TestTexasPokerEventLogsIncompleteDPZXAndRejectsUnassignedSources(t *testing.T) {
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

	now := time.Unix(1792526400, 0)
	dpzxBody := `{"packageName":"com.dpzx.pro","timestamp":1792526400000,` +
		`"cmd":"401","direction":"ServerToClient","location":"TABLE",` +
		`"data":{"marker":"dpzx-private-hand"}}`
	var dpzxContext app.RequestContext
	dpzxContext.Request.SetBodyString(dpzxBody)
	texasPokerEventAt(context.Background(), &dpzxContext, now)
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
			texasPokerEventAt(context.Background(), &requestContext, now)
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
		ctx := postAAPokerTestEvent(t, now, commands[i], events[i])
		if ctx.Response.StatusCode() != consts.StatusOK {
			t.Fatalf("setup event %d status = %d", i, ctx.Response.StatusCode())
		}
		now = now.Add(time.Millisecond)
	}

	failed := postAAPokerTestEvent(t, now, commands[4], events[4])
	if failed.Response.StatusCode() != consts.StatusServiceUnavailable || calls != 1 {
		t.Fatalf("temporary failure status/calls = %d/%d", failed.Response.StatusCode(), calls)
	}
	now = now.Add(time.Millisecond)
	succeeded := postAAPokerTestEvent(t, now, commands[4], events[4])
	if succeeded.Response.StatusCode() != consts.StatusOK ||
		!strings.Contains(string(succeeded.Response.Body()), `"Result":"Call 8"`) || calls != 2 {
		t.Fatalf("retry response/calls = %s/%d", succeeded.Response.Body(), calls)
	}
	now = now.Add(time.Millisecond)
	replayed := postAAPokerTestEvent(t, now, commands[4], events[4])
	if replayed.Response.StatusCode() != consts.StatusOK ||
		!strings.Contains(string(replayed.Response.Body()), `"Result":"Call 8"`) || calls != 2 {
		t.Fatalf("cached replay/calls = %s/%d", replayed.Response.Body(), calls)
	}
}

func postAAPokerTestEvent(
	t *testing.T, now time.Time, command, data string,
) *app.RequestContext {
	t.Helper()
	body := fmt.Sprintf(`{"packageName":"com.plusaa.amula","uuid":"user@example.com","vpip":0,"pfr":0,"timestamp":%d,"uid":1762080,"cmd":%q,"direction":"ServerToClient","location":"TABLE","dataExtra":99,"data":%s}`,
		now.UnixMilli(), command, data)
	requestContext := &app.RequestContext{}
	requestContext.Request.SetBodyString(body)
	requestContext.Request.Header.Set("Authorization", "Bearer session-token")
	texasPokerEventAt(context.Background(), requestContext, now)
	return requestContext
}
