package handler

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	ddbaby "github.com/kingjxu/ddbaby/biz/model/ddbaby"
	"github.com/kingjxu/ddbaby/service"
	"github.com/sirupsen/logrus"
)

const (
	aapokerPackageName          = "com.plusaa.amula"
	aapokerAlternatePackageName = "com.ty.aapk"
	dpzxPackageName             = "com.dpzx.pro"
)

// Use the same middleware chain as the generated HTTP route.
func serveTexasPokerEventForTest(c *app.RequestContext) {
	c.SetHandlers(app.HandlersChain{PrepareTexasPokerEvent, TexasPokerEvent})
	c.SetIndex(-1)
	c.Next(context.Background())
}

// Exercise the production binder while allowing parse assertions to inspect raw tokens.
func bindTexasPokerTestEvent(body []byte) (*ddbaby.TexasPokerEventReq, error) {
	var c app.RequestContext
	c.Request.SetMethod("POST")
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.SetBody(body)
	c.Request.Header.SetContentLength(len(body))
	var req ddbaby.TexasPokerEventReq
	if err := c.BindAndValidate(&req); err != nil {
		return nil, err
	}
	if !service.IsSupportedPokerEventPackage(req.PackageName) {
		return nil, fmt.Errorf("invalid packageName")
	}
	return &req, nil
}

func TestTexasPokerEventHandlerHandlesWithoutHTTPContext(t *testing.T) {
	for _, test := range []struct {
		name     string
		req      *ddbaby.TexasPokerEventReq
		wantCode int32
	}{
		{name: "missing request", wantCode: 400},
		{name: "unknown package", req: &ddbaby.TexasPokerEventReq{PackageName: "unknown"}, wantCode: 400},
		{name: "accepted event", req: &ddbaby.TexasPokerEventReq{PackageName: dpzxPackageName}},
	} {
		t.Run(test.name, func(t *testing.T) {
			handler := NewTexasPokerEventHandler(test.req)
			handler.rawBody = []byte(`{"marker":"handler-test"}`)
			response, err := handler.Handle(context.Background())
			if (err != nil) != (test.wantCode != 0) {
				t.Fatalf("Handle error = %v", err)
			}
			if response == nil || response.BaseResp == nil || response.BaseResp.StatusCode != test.wantCode {
				t.Fatalf("Handle response = %+v", response)
			}
			if response.Result != nil {
				t.Fatal("basic event unexpectedly produced a decision")
			}
		})
	}
}

func TestBindTexasPokerEvent(t *testing.T) {
	body := []byte(`{"packageName":"com.plusaa.amula","timestamp":1792526400000,"pid":"42","uid":9007199254740993,` +
		`"PacketId":123,"cmd":"RSP_GAME_STATE","direction":"ServerToClient",` +
		`"data":{"table":7},"location":"TABLE","seq":1,"dataExtra":"7"}`)
	event, err := bindTexasPokerTestEvent(body)
	if err != nil {
		t.Fatalf("bindTexasPokerTestEvent() error = %v", err)
	}
	if event.Cmd != "RSP_GAME_STATE" || event.Direction != "ServerToClient" || event.Location != "TABLE" {
		t.Fatalf("unexpected event: %+v", event)
	}
	if string(event.UID) != "9007199254740993" || string(event.Data) != `{"table":7}` {
		t.Fatalf("dynamic JSON changed: uid=%s data=%s", event.UID, event.Data)
	}
}

func TestBindTexasPokerEventAcceptsAAPokerPackageAliases(t *testing.T) {
	for _, packageName := range []string{aapokerPackageName, aapokerAlternatePackageName} {
		body := fmt.Sprintf(`{"packageName":%q,"timestamp":1,"cmd":"STATE",`+
			`"direction":"ServerToClient","location":"TABLE","data":{}}`, packageName)
		event, err := bindTexasPokerTestEvent([]byte(body))
		if err != nil {
			t.Fatalf("package %q rejected: %v", packageName, err)
		}
		if event.PackageName != packageName {
			t.Fatalf("package changed: %q", event.PackageName)
		}
	}
}

func TestTexasPokerEventHTTPResponse(t *testing.T) {
	validBody := `{"packageName":"com.plusaa.amula","timestamp":1792526400000,"cmd":"STATE",` +
		`"direction":"ServerToClient","location":"TABLE","data":{}}`
	var validContext app.RequestContext
	validContext.Request.SetMethod("POST")
	validContext.Request.Header.Set("Content-Type", "application/json")
	validContext.Request.SetBodyString(validBody)
	validContext.Request.Header.SetContentLength(len(validBody))
	serveTexasPokerEventForTest(&validContext)
	if validContext.Response.StatusCode() != consts.StatusOK {
		t.Fatalf("valid status = %d, want 200", validContext.Response.StatusCode())
	}
	if !strings.Contains(string(validContext.Response.Body()), `"StatusCode":0`) {
		t.Fatalf("valid response = %s", validContext.Response.Body())
	}

	var invalidContext app.RequestContext
	invalidContext.Request.SetMethod("POST")
	invalidContext.Request.Header.Set("Content-Type", "application/json")
	invalidContext.Request.SetBodyString(`{"timestamp":1}`)
	invalidContext.Request.Header.SetContentLength(len(`{"timestamp":1}`))
	serveTexasPokerEventForTest(&invalidContext)
	if invalidContext.Response.StatusCode() != consts.StatusBadRequest {
		t.Fatalf("invalid status = %d, want 400", invalidContext.Response.StatusCode())
	}
}

func TestTexasPokerEventMiddlewarePreservesRequestContext(t *testing.T) {
	for _, body := range []string{
		`{ "packageName": "com.dpzx.pro", "data": { "value": 9007199254740993 } }`,
		`{"packageName":"com.dpzx.pro","timestamp":"invalid"}`,
	} {
		var c app.RequestContext
		c.Request.SetMethod("POST")
		c.Request.SetRequestURI("/lyxz/gto_event?PackageName=com.plusaa.amula&Cmd=ignored")
		c.Request.Header.Set("Content-Type", "application/json")
		c.Request.SetBodyString(body)
		serveTexasPokerEventForTest(&c)
		if string(c.Request.Body()) != body || string(c.Request.URI().QueryString()) != "PackageName=com.plusaa.amula&Cmd=ignored" {
			t.Fatal("middleware did not preserve the raw body and original query")
		}
	}
}

func TestTexasPokerEventBindsBufferedJSON(t *testing.T) {
	body := `{"packageName":"com.dpzx.pro","cmd":"401","data":{"marker":"stream-event"}}`
	var c app.RequestContext
	c.Request.SetMethod("POST")
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.SetBodyString(body)
	c.Request.Header.SetContentLength(len(body))
	var logs bytes.Buffer
	previousOutput := logrus.StandardLogger().Out
	logrus.SetOutput(&logs)
	t.Cleanup(func() { logrus.SetOutput(previousOutput) })
	serveTexasPokerEventForTest(&c)
	if c.Response.StatusCode() != consts.StatusOK || !strings.Contains(logs.String(), "stream-event") {
		t.Fatalf("stream response/log = %s / %s", c.Response.Body(), logs.String())
	}
}

func TestTexasPokerEventRejectsTypedFieldMismatchBeforeLogging(t *testing.T) {
	var c app.RequestContext
	c.Request.SetMethod("POST")
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.SetBodyString(`{"packageName":"com.dpzx.pro","timestamp":"private-invalid-marker"}`)
	var logs bytes.Buffer
	previousOutput := logrus.StandardLogger().Out
	logrus.SetOutput(&logs)
	t.Cleanup(func() { logrus.SetOutput(previousOutput) })
	serveTexasPokerEventForTest(&c)
	if c.Response.StatusCode() != consts.StatusBadRequest || strings.Contains(logs.String(), "private-invalid-marker") ||
		strings.Contains(string(c.Response.Body()), "private-invalid-marker") {
		t.Fatalf("invalid response/log = %s / %s", c.Response.Body(), logs.String())
	}
}

func TestTexasPokerEventDoesNotRateLimitAndLogsEveryPayload(t *testing.T) {
	const requestCount = 301
	data := strings.Repeat("a", 4*1024)
	body := `{"packageName":"com.plusaa.amula","timestamp":1792526400000,"cmd":"STATE",` +
		`"direction":"ServerToClient","location":"TABLE","data":"` + data + `"}`

	var logs bytes.Buffer
	previousOutput := logrus.StandardLogger().Out
	logrus.SetOutput(&logs)
	t.Cleanup(func() { logrus.SetOutput(previousOutput) })

	statuses := make([]int, 0, requestCount)
	for i := 0; i < requestCount; i++ {
		var requestContext app.RequestContext
		requestContext.Request.SetMethod("POST")
		requestContext.Request.Header.Set("Content-Type", "application/json")
		requestContext.Request.SetBodyString(body)
		requestContext.Request.Header.SetContentLength(len(body))
		serveTexasPokerEventForTest(&requestContext)
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

func TestBindTexasPokerEventRejectsInvalidInput(t *testing.T) {
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
			if _, err := bindTexasPokerTestEvent([]byte(body)); err == nil {
				t.Fatal("expected invalid event to be rejected")
			}
		})
	}
}

func TestBindTexasPokerEventValidatesTypedFields(t *testing.T) {
	body := []byte(`{"packageName":"com.dpzx.pro","timestamp":"unknown","pid":{},` +
		`"uid":true,"cmd":{"new":"shape"},"direction":7,"location":false}`)
	if _, err := bindTexasPokerTestEvent(body); err == nil {
		t.Fatal("typed field mismatch must be rejected")
	}

	largeData := `"` + strings.Repeat("a", 490*1024) + `"`
	largeBody := `{"packageName":"com.dpzx.pro","data":` + largeData + `}`
	if len(largeBody) >= 4*1024*1024 {
		t.Fatalf("test body unexpectedly exceeds total body limit: %d", len(largeBody))
	}
	if _, err := bindTexasPokerTestEvent([]byte(largeBody)); err != nil {
		t.Fatalf("data above former 480 KiB sub-limit rejected: %v", err)
	}
}

func TestTexasPokerEventLogsIncompleteDPZXAndRejectsUnassignedSources(t *testing.T) {
	auth := failEventDecision(t)

	var logs bytes.Buffer
	previousOutput := logrus.StandardLogger().Out
	logrus.SetOutput(&logs)
	t.Cleanup(func() { logrus.SetOutput(previousOutput) })

	dpzxBody := `{"packageName":"com.dpzx.pro","timestamp":1792526400000,` +
		`"cmd":"401","direction":"ServerToClient","location":"TABLE",` +
		`"data":{"marker":"dpzx-private-hand"}}`
	var dpzxContext app.RequestContext
	dpzxContext.Request.SetMethod("POST")
	dpzxContext.Request.Header.Set("Content-Type", "application/json")
	dpzxContext.Request.SetBodyString(dpzxBody)
	dpzxContext.Request.Header.SetContentLength(len(dpzxBody))
	serveTexasPokerEventForTest(&dpzxContext)
	if dpzxContext.Response.StatusCode() != consts.StatusOK ||
		strings.Contains(string(dpzxContext.Response.Body()), `"Result"`) {
		t.Fatalf("DPZX response = %d %s", dpzxContext.Response.StatusCode(), dpzxContext.Response.Body())
	}
	if auth.calls.Load() != 0 {
		t.Fatal("incomplete DPZX event entered decision authentication")
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
			requestContext.Request.SetMethod("POST")
			requestContext.Request.Header.Set("Content-Type", "application/json")
			requestContext.Request.SetBodyString(body)
			requestContext.Request.Header.SetContentLength(len(body))
			serveTexasPokerEventForTest(&requestContext)
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
	active := useEventActiveCodeDB(t)
	auth := failEventDecision(t)
	tableID := time.Now().UnixNano()

	events := []string{
		`{"mySeatID":0,"playerIDArray":[1762080,2836313,2646258,2838496,2859796,-1,2860755,1081977],"operationID":4,"operationRound":16}`,
		`{"bankerID":0,"operationID":4,"firstCardArray":[12,-1,-1,-1,-1,-1,-1,-1],"SecondCardArray":[24,-1,-1,-1,-1,-1,-1,-1],"smallChip":2,"bigChip":4,"chipsArray":[762,996,392,1557,606,-1,768,1020],"canPlayStatus":[1,1,1,1,1,0,1,1],"seatStatus":[2,2,2,2,2,0,2,2],"handNum":3,"straddlePop":[0,0,0,8,0,0,0,0],"operationRound":17,"mushroomPool":40,"pool_chips":42,"anteCountArray":[4,6,8,12,4,0,4,4]}`,
		`{"seatID":4,"action":6,"operationID":6,"anteNumber":0,"operationRound":18}`,
		`{"seatID":6,"action":2,"operationID":7,"anteNumber":8,"operationRound":19}`,
		`{"seatID":7,"action":2,"operationID":0,"anteNumber":8,"operationRound":20}`,
	}
	commands := []string{"REQ_GAME_ENTER_ROOM", "REQ_GAME_RECV_START_INFOR", "REQ_GAME_RECV_ACTION", "REQ_GAME_RECV_ACTION", "REQ_GAME_RECV_ACTION"}
	now := time.Unix(1792526400, 0)
	for i := 0; i < len(events)-1; i++ {
		ctx := postAAPokerTestEvent(t, tableID, now, commands[i], events[i])
		if ctx.Response.StatusCode() != consts.StatusOK {
			t.Fatalf("setup event %d status = %d", i, ctx.Response.StatusCode())
		}
		now = now.Add(time.Millisecond)
	}

	failed := postAAPokerTestEvent(t, tableID, now, commands[4], events[4])
	if failed.Response.StatusCode() != consts.StatusServiceUnavailable || auth.calls.Load() != 1 {
		t.Fatalf("temporary failure status/calls = %d/%d", failed.Response.StatusCode(), auth.calls.Load())
	}
	retried := postAAPokerTestEvent(t, tableID, now, commands[4], events[4])
	if retried.Response.StatusCode() != consts.StatusServiceUnavailable || auth.calls.Load() != 2 {
		t.Fatalf("failed decision did not release retry eligibility: status=%d calls=%d", retried.Response.StatusCode(), auth.calls.Load())
	}
	req, err := bindTexasPokerTestEvent(retried.Request.Body())
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := service.ProcessPokerEvent(req)
	if err != nil || candidate == nil || candidate.Cached != nil {
		t.Fatalf("retry candidate = %+v, %v", candidate, err)
	}
	// Simulate a completed strategy result through the service lifecycle API;
	// subsequent HTTP requests must replay it without any new side effects.
	service.CompletePokerEventDecision(candidate, &service.PokerEventDecisionResult{
		Result: "Call 8", HeroCard: []string{"A-D", "K-C"},
		DecisionAction: "call", DecisionAmount: 8, HeroMaxTarget: 8,
	})
	now = now.Add(time.Millisecond)
	succeeded := postAAPokerTestEvent(t, tableID, now, commands[4], events[4])
	if succeeded.Response.StatusCode() != consts.StatusOK ||
		!strings.Contains(string(succeeded.Response.Body()), `"Result":"Call 8"`) || auth.calls.Load() != 2 {
		t.Fatalf("retry response/calls = %s/%d", succeeded.Response.Body(), auth.calls.Load())
	}
	active.info.ExpireAt = time.Now().Add(-time.Hour).Unix()
	blocked := postAAPokerTestEvent(t, tableID, now, commands[4], events[4])
	if blocked.Response.StatusCode() != consts.StatusServiceUnavailable {
		t.Fatal("expired activation replayed cached decision")
	}
	active.info.ExpireAt = time.Now().Add(time.Hour).Unix()
	now = now.Add(time.Millisecond)
	replayed := postAAPokerTestEvent(t, tableID, now, commands[4], events[4])
	if replayed.Response.StatusCode() != consts.StatusOK ||
		!strings.Contains(string(replayed.Response.Body()), `"Result":"Call 8"`) || auth.calls.Load() != 2 {
		t.Fatalf("cached replay/calls = %s/%d", replayed.Response.Body(), auth.calls.Load())
	}
}

func postAAPokerTestEvent(
	t *testing.T, tableID int64, now time.Time, command, data string,
) *app.RequestContext {
	t.Helper()
	body := fmt.Sprintf(`{"packageName":"com.plusaa.amula","uuid":"user@example.com","vpip":0,"pfr":0,"timestamp":%d,"uid":1762080,"cmd":%q,"direction":"ServerToClient","location":"TABLE","dataExtra":%d,"data":%s}`,
		now.UnixMilli(), command, tableID, data)
	requestContext := &app.RequestContext{}
	requestContext.Request.SetMethod("POST")
	requestContext.Request.Header.Set("Content-Type", "application/json")
	requestContext.Request.SetBodyString(body)
	requestContext.Request.Header.SetContentLength(len(body))
	serveTexasPokerEventForTest(requestContext)
	return requestContext
}
