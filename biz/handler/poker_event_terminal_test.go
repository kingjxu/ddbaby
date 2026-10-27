package handler

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	_const "github.com/kingjxu/ddbaby/const"
)

// 非法结果只终止当前行动点：重复事件不重试，后续实际行动仍可生成合法决策。
func TestInvalidEventDecisionEndsOnlyCurrentTurn(t *testing.T) {
	aapokerEvents := []string{
		`{"mySeatID":0,"playerIDArray":[1762080,2836313,2646258,2838496,2859796,-1,2860755,1081977],"operationID":4,"operationRound":16}`,
		`{"bankerID":0,"operationID":4,"firstCardArray":[12,-1,-1,-1,-1,-1,-1,-1],"SecondCardArray":[24,-1,-1,-1,-1,-1,-1,-1],"smallChip":2,"bigChip":4,"chipsArray":[762,996,392,1557,606,-1,768,1020],"canPlayStatus":[1,1,1,1,1,0,1,1],"seatStatus":[2,2,2,2,2,0,2,2],"handNum":3,"straddlePop":[0,0,0,8,0,0,0,0],"operationRound":17,"mushroomPool":40,"pool_chips":42,"anteCountArray":[4,6,8,12,4,0,4,4]}`,
		`{"seatID":4,"action":6,"operationID":6,"anteNumber":0,"operationRound":18}`,
		`{"seatID":6,"action":2,"operationID":7,"anteNumber":8,"operationRound":19}`,
		`{"seatID":7,"action":2,"operationID":0,"anteNumber":8,"operationRound":20}`,
	}
	aapokerCommands := []string{"REQ_GAME_ENTER_ROOM", "REQ_GAME_RECV_START_INFOR", "REQ_GAME_RECV_ACTION", "REQ_GAME_RECV_ACTION", "REQ_GAME_RECV_ACTION"}

	for _, source := range []string{aapokerPackageName, dpzxPackageName} {
		t.Run(source, func(t *testing.T) {
			db := useEventActiveCodeDB(t)
			db.info.CodeType = _const.ActiveCodeTypeByInvokeCnt
			db.info.InvokeCnt, db.info.TotalCnt = 0, 10
			strategy := failEventDecision(t)
			strategy.success = true
			strategy.action = "check"
			uuid := fmt.Sprintf("rejected-%d", time.Now().UnixNano())
			post := func(body string) *app.RequestContext {
				t.Helper()
				c := &app.RequestContext{}
				c.Request.SetMethod("POST")
				c.Request.Header.Set("Content-Type", "application/json")
				c.Request.SetBodyString(body)
				c.Request.Header.SetContentLength(len(body))
				serveTexasPokerEventForTest(c)
				return c
			}
			var bodies []string
			if source == aapokerPackageName {
				table := time.Now().UnixNano()
				for i, data := range aapokerEvents {
					bodies = append(bodies, fmt.Sprintf(`{"packageName":%q,"uuid":%q,"timestamp":%d,"uid":1762080,"cmd":%q,"direction":"ServerToClient","location":"TABLE","dataExtra":%d,"data":%s}`, source, uuid, 1000+i, aapokerCommands[i], table, data))
				}
			} else {
				steps := []struct{ cmd, data string }{
					{"430", dpzxRealRoomSnapshot}, {"431", dpzxRealHandStart},
					{"432", `[2,15,0,100,true]`},
					{"492", `[false,[{"index":3,"vecCards":[{"value":8},{"value":43}]}]]`},
					{"433", `[{"index":2,"betState":2,"score":5904,"betted":100}]`},
					{"403", `[425]`}, {"432", `[3,15,0,100,true]`},
				}
				for i, step := range steps {
					bodies = append(bodies, fmt.Sprintf(`{"packageName":%q,"uuid":%q,"timestamp":%d,"uid":2773921,"cmd":%q,"direction":"ServerToClient","location":"TABLE","data":%s}`, source, uuid, 1000+i, step.cmd, step.data))
				}
			}
			for _, body := range bodies {
				c := post(body)
				if c.Response.StatusCode() != 200 {
					t.Fatalf("setup failed: %s", c.Response.Body())
				}
			}
			assertRejected := func(body string) {
				t.Helper()
				c := post(body)
				if c.Response.StatusCode() != 200 || !strings.Contains(string(c.Response.Body()), "decision_rejected") || strings.Contains(string(c.Response.Body()), `"Result"`) {
					t.Fatalf("invalid turn was not terminated: %d %s", c.Response.StatusCode(), c.Response.Body())
				}
			}
			assertRejected(bodies[len(bodies)-1])
			select {
			case audit := <-db.audits:
				if string(audit.FinalResponse) != "null" {
					t.Fatalf("invalid decision delivered: %+v", audit)
				}
			case <-time.After(time.Second):
				t.Fatal("missing rejected decision audit")
			}
			if strategy.calls.Load() != 1 || db.billings != 0 {
				t.Fatal("rejection retried strategy or billed")
			}
			var next []string
			if source == aapokerPackageName {
				steps := []string{
					`{"FirstCard":-1,"SecondCard":-1,"seatID":0,"action":2,"operationID":1,"anteNumber":8,"operationRound":21}`,
					`{"seatID":1,"action":6,"operationID":2,"anteNumber":0,"operationRound":22}`,
					`{"seatID":2,"action":6,"operationID":3,"anteNumber":0,"operationRound":23}`,
					`{"seatID":3,"action":6,"operationID":6,"anteNumber":0,"operationRound":24}`,
					`{"seatID":6,"action":6,"operationID":7,"anteNumber":0,"operationRound":25}`,
					`{"seatID":7,"action":3,"operationID":0,"anteNumber":8,"operationRound":26}`,
				}
				// Keep the original identity/table envelope; replace only the action and timestamp.
				for i, data := range steps {
					prefix := bodies[len(bodies)-1][:strings.Index(bodies[len(bodies)-1], `"data":`)]
					prefix = strings.Replace(prefix, `"timestamp":1004`, fmt.Sprintf(`"timestamp":%d`, 2000+i), 1)
					next = append(next, prefix+`"data":`+data+`}`)
				}
			} else {
				steps := []struct{ cmd, data string }{
					{"433", `[{"index":3,"betState":2,"score":4875,"betted":100}]`},
					{"432", `[4,15,0,100,true]`},
					{"433", `[{"index":4,"betState":6,"score":5000,"betted":0}]`},
					{"432", `[5,15,0,100,true]`},
					{"433", `[{"index":5,"betState":6,"score":5000,"betted":0}]`},
					{"432", `[6,15,0,100,true]`},
					{"433", `[{"index":6,"betState":6,"score":5000,"betted":0}]`},
					{"432", `[7,15,0,100,true]`},
					{"433", `[{"index":7,"betState":6,"score":5000,"betted":0}]`},
					{"432", `[1,15,0,100,true]`},
					{"433", `[{"index":1,"betState":6,"score":5000,"betted":100}]`},
					{"432", `[2,15,0,100,true]`},
					{"433", `[{"index":2,"betState":4,"score":5704,"betted":300}]`},
					{"432", `[3,15,100,200,true]`},
				}
				for i, step := range steps {
					next = append(next, fmt.Sprintf(`{"packageName":%q,"uuid":%q,"timestamp":%d,"uid":2773921,"cmd":%q,"direction":"ServerToClient","location":"TABLE","data":%s}`, source, uuid, 2000+i, step.cmd, step.data))
				}
			}
			strategy.action = "call"
			var last *app.RequestContext
			for _, body := range next {
				last = post(body)
				if last.Response.StatusCode() != 200 {
					t.Fatalf("later action blocked: %s", last.Response.Body())
				}
			}
			if !strings.Contains(string(last.Response.Body()), `"Result"`) || strategy.calls.Load() != 2 || db.billings != 1 {
				t.Fatalf("next turn did not decide: %s strategy=%d billings=%d", last.Response.Body(), strategy.calls.Load(), db.billings)
			}
			select {
			case <-db.audits:
			case <-time.After(time.Second):
				t.Fatal("missing next turn audit")
			}
			select {
			case <-db.audits:
				t.Fatal("duplicate audit")
			default:
			}
		})
	}
}
