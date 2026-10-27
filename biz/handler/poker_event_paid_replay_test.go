package handler

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	_const "github.com/kingjxu/ddbaby/const"
	"github.com/kingjxu/ddbaby/service"
)

// Runs the real handler, strategy orchestration, quota update and asynchronous
// audit path against test DB callbacks and an HTTP strategy response.
func TestLastPaidEventIsRejectedAfterQuotaExhaustion(t *testing.T) {
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
			db.info.InvokeCnt, db.info.TotalCnt = 9, 10
			strategy := failEventDecision(t)
			strategy.success = true
			uuid := fmt.Sprintf("paid-%d", time.Now().UnixNano())
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
			var success *app.RequestContext
			for _, body := range bodies {
				success = post(body)
				if success.Response.StatusCode() != 200 {
					t.Fatalf("setup/decision failed: %s", success.Response.Body())
				}
			}
			if !strings.Contains(string(success.Response.Body()), `"Result"`) || strategy.calls.Load() != 1 || db.billings != 1 || db.info.InvokeCnt != 10 {
				t.Fatalf("last decision not charged once: %s strategy=%d billings=%d quota=%d", success.Response.Body(), strategy.calls.Load(), db.billings, db.info.InvokeCnt)
			}
			select {
			case <-db.audits:
			case <-time.After(time.Second):
				t.Fatal("missing decision audit")
			}
			// Discard the first response and retry the identical event.
			body := bodies[len(bodies)-1]
			req, err := bindTexasPokerTestEvent([]byte(body))
			if err != nil {
				t.Fatal(err)
			}
			cached, err := service.ProcessPokerEvent(req)
			if err != nil || cached == nil || cached.Cached == nil {
				t.Fatalf("successful result was not cached: %+v %v", cached, err)
			}
			for i := 0; i < 2; i++ {
				replay := post(body)
				if replay.Response.StatusCode() != 503 || strings.Contains(string(replay.Response.Body()), `"Result"`) {
					t.Fatalf("exhausted quota replayed paid result: %d %s", replay.Response.StatusCode(), replay.Response.Body())
				}
			}
			// Another event/identity must not retrieve the paid response or consume state.
			for name, altered := range map[string]string{
				"next event":   strings.Replace(body, `"timestamp":`, `"timestamp":9`, 1),
				"other uuid":   strings.Replace(body, uuid, uuid+"-other", 1),
				"other source": strings.Replace(body, source, "com.ty.aapk", 1),
			} {
				rejected := post(altered)
				if rejected.Response.StatusCode() != 503 {
					t.Fatalf("%s bypassed quota: %s", name, rejected.Response.Body())
				}
			}
			if strategy.calls.Load() != 1 || db.billings != 1 {
				t.Fatal("replay repeated strategy or billing")
			}
			select {
			case <-db.audits:
				t.Fatal("replay repeated audit")
			default:
			}
			db.err = fmt.Errorf("test database unavailable")
			if post(body).Response.StatusCode() != 503 {
				t.Fatal("database failure bypassed activation")
			}
		})
	}
}
