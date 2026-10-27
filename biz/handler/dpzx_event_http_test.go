package handler

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/kingjxu/ddbaby/service"
)

const dpzxRealRoomSnapshot = `[{"bankIndex":6,"mySeat":3,"HandNum":125,"roomID":28815717,"littleBB":50,"ante":25,"stisNumMax":8},[{"index":1,"value":25,"score":7100},{"index":2,"value":25,"score":6004},{"index":3,"value":25,"score":4975},{"index":4,"value":25,"score":17545},{"index":5,"value":25,"score":14975},{"index":6,"value":25,"score":4413},{"index":7,"value":25,"score":3681}]]`
const dpzxRealHandStart = `[[{"index":1,"value":100,"score":7000,"startState":2,"betState":10},{"index":2,"value":0,"score":6004,"startState":0,"betState":10},{"index":3,"value":0,"score":4975,"startState":0,"betState":10},{"index":4,"value":0,"score":17545,"startState":0,"betState":10},{"index":5,"value":0,"score":14975,"startState":0,"betState":10},{"index":6,"value":0,"score":4413,"startState":0,"betState":10},{"index":7,"value":50,"score":3631,"startState":1,"betState":10}]]`

func TestTexasPokerEventRoutesDPZXThroughDecisionResponse(t *testing.T) {
	useEventActiveCodeDB(t)
	auth := failEventDecision(t)
	uuid := fmt.Sprintf("hero-%d@example.com", time.Now().UnixNano())

	post := func(command, payload string, timestamp int64) *app.RequestContext {
		t.Helper()
		body := fmt.Sprintf(`{"packageName":"com.dpzx.pro","uuid":%q,"vpip":0,"pfr":0,"timestamp":%d,"uid":2773921,"cmd":%q,"direction":"ServerToClient","location":"TABLE","data":%s}`,
			uuid, timestamp, command, payload)
		requestContext := &app.RequestContext{}
		requestContext.Request.SetMethod("POST")
		requestContext.Request.Header.Set("Content-Type", "application/json")
		requestContext.Request.SetBodyString(body)
		requestContext.Request.Header.SetContentLength(len(body))
		serveTexasPokerEventForTest(requestContext)
		return requestContext
	}

	post("430", dpzxRealRoomSnapshot, 1793076613000)
	post("431", dpzxRealHandStart, 1793076615000)
	post("432", `[2,15,0,100,true]`, 1793076615500)
	post("492", `[false,[{"index":3,"vecCards":[{"value":8},{"value":43}]}]]`, 1793076615600)
	post("433", `[{"index":2,"betState":2,"score":5904,"betted":100}]`, 1793076617000)
	post("403", `[425]`, 1793076617100)
	failed := post("432", `[3,15,0,100,true]`, 1793076617478)
	if failed.Response.StatusCode() != consts.StatusServiceUnavailable || auth.calls.Load() != 1 {
		t.Fatalf("failed response/calls = %s/%d", failed.Response.Body(), auth.calls.Load())
	}
	retried := post("432", `[3,15,0,100,true]`, 1793076617478)
	if retried.Response.StatusCode() != consts.StatusServiceUnavailable || auth.calls.Load() != 2 {
		t.Fatalf("retry status/calls = %d/%d", retried.Response.StatusCode(), auth.calls.Load())
	}
	req, err := bindTexasPokerTestEvent(retried.Request.Body())
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := service.ProcessPokerEvent(req)
	if err != nil || candidate == nil || candidate.Cached != nil {
		t.Fatalf("retry candidate = %+v, %v", candidate, err)
	}
	if req.UUID != uuid || candidate.Request.CurrentPlayerPosition != "UTG+1-Hero" {
		t.Fatalf("unexpected candidate: %+v", candidate)
	}
	service.CompletePokerEventDecision(candidate, &service.PokerEventDecisionResult{
		Result: "Call 100", HeroCard: []string{"10-H", "6-S"},
		DecisionAction: "call", DecisionAmount: 0, HeroMaxTarget: 4975,
	})
	decision := post("432", `[3,15,0,100,true]`, 1793076617478)
	if decision.Response.StatusCode() != consts.StatusOK ||
		!strings.Contains(string(decision.Response.Body()), `"Result":"Call 100"`) || auth.calls.Load() != 2 {
		t.Fatalf("decision response/calls = %s/%d", decision.Response.Body(), auth.calls.Load())
	}
	replayed := post("432", `[3,15,0,100,true]`, 1793076617478)
	if replayed.Response.StatusCode() != consts.StatusOK ||
		!strings.Contains(string(replayed.Response.Body()), `"Result":"Call 100"`) || auth.calls.Load() != 2 {
		t.Fatalf("replay response/calls = %s/%d", replayed.Response.Body(), auth.calls.Load())
	}
}
