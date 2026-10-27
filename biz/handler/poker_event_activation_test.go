package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	ddbaby "github.com/kingjxu/ddbaby/biz/model/ddbaby"
	_const "github.com/kingjxu/ddbaby/const"
	dbmysql "github.com/kingjxu/ddbaby/dal/mysql"
	poker "github.com/kingjxu/ddbaby/dal/mysql/texas_poker"
	"github.com/kingjxu/ddbaby/model"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Fail at the real HTTP boundary without making external requests or replacing
// the business function under test.
type eventDecisionFailureTransport struct {
	t       *testing.T
	calls   atomic.Int32
	success bool
	action  string
}

func (transport *eventDecisionFailureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	transport.calls.Add(1)
	if req.URL.Path != "/gto/decision" {
		transport.t.Errorf("unexpected outbound request: path=%q", req.URL.Path)
	}
	if transport.success {
		action := transport.action
		if action == "" {
			action = "call"
		}
		data, _ := json.Marshal(model.TexasGtoDecisionResp{Action: action, ActionProbability: 1})
		return &http.Response{StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(string(data))), Header: make(http.Header)}, nil
	}
	return &http.Response{StatusCode: http.StatusServiceUnavailable,
		Body: io.NopCloser(strings.NewReader(`{"error":"temporary failure"}`)), Header: make(http.Header)}, nil
}

func failEventDecision(t *testing.T) *eventDecisionFailureTransport {
	t.Helper()
	t.Setenv("GTO_API_KEY", "unit-test-key")
	transport := &eventDecisionFailureTransport{t: t}
	previous := http.DefaultTransport
	http.DefaultTransport = transport
	t.Cleanup(func() { http.DefaultTransport = previous })
	return transport
}

func TestEventActivationCheckedBeforeLoggingAndState(t *testing.T) {
	for _, packageName := range []string{aapokerPackageName, dpzxPackageName} {
		for _, test := range []struct {
			name string
			info *poker.UserActiveCode
			err  error
		}{
			{name: "missing"},
			{name: "expired", info: &poker.UserActiveCode{CodeType: _const.ActiveCodeTypeByExpireAt, ExpireAt: time.Now().Add(-time.Hour).Unix()}},
			{name: "exhausted", info: &poker.UserActiveCode{CodeType: _const.ActiveCodeTypeByInvokeCnt, InvokeCnt: 10, TotalCnt: 10}},
			{name: "database error", err: errors.New("test database unavailable")},
		} {
			t.Run(packageName+"/"+test.name, func(t *testing.T) {
				db := useEventActiveCodeDB(t)
				db.info, db.err = test.info, test.err
				outbound := failEventDecision(t)
				var logs bytes.Buffer
				previous := logrus.StandardLogger().Out
				logrus.SetOutput(&logs)
				t.Cleanup(func() { logrus.SetOutput(previous) })
				h := NewTexasPokerEventHandler(&ddbaby.TexasPokerEventReq{
					PackageName: packageName, UUID: "inactive-user", Cmd: "430",
				})
				h.rawBody = []byte("private-event-marker")
				resp, err := h.Handle(context.Background())
				if err == nil || resp.BaseResp.StatusCode != 503 || (db.calls != 1 || len(db.users) != 1 || db.users[0] != "inactive-user") || outbound.calls.Load() != 0 {
					t.Fatalf("activation gate: response=%+v err=%v db=%d outbound=%d", resp, err, db.calls, outbound.calls.Load())
				}
				if strings.Contains(logs.String(), "private-event-marker") || strings.Contains(logs.String(), "state update") {
					t.Fatal("activation failure proceeded to logging/state processing")
				}
			})
		}
	}
}

func TestEventActivationAcceptsAvailableQuotaWithoutToken(t *testing.T) {
	for _, info := range []*poker.UserActiveCode{
		{CodeType: _const.ActiveCodeTypeByExpireAt, ExpireAt: time.Now().Add(time.Hour).Unix()},
		{CodeType: _const.ActiveCodeTypeByInvokeCnt, InvokeCnt: 9, TotalCnt: 10},
	} {
		db := useEventActiveCodeDB(t)
		db.info = info
		h := NewTexasPokerEventHandler(&ddbaby.TexasPokerEventReq{
			PackageName: dpzxPackageName, UUID: "active-user", Cmd: "401",
		})
		resp, err := h.Handle(context.Background())
		if err != nil || resp.BaseResp.StatusCode != 0 || db.calls != 1 || db.users[0] != "active-user" {
			t.Fatalf("valid activation rejected: response=%+v err=%v queries=%d", resp, err, db.calls)
		}
	}
}

// Replace only GORM's query callback in this test database. Production still
// uses verifyActiveCode and the real DAL; no production function is replaced.
type eventActiveCodeDB struct {
	info     *poker.UserActiveCode
	err      error
	calls    int
	users    []string
	billings int
	audits   chan *poker.GtoDecisionRecord
}

func useEventActiveCodeDB(t *testing.T) *eventActiveCodeDB {
	t.Helper()
	state := &eventActiveCodeDB{info: &poker.UserActiveCode{
		CodeType: _const.ActiveCodeTypeByExpireAt, ExpireAt: time.Now().Add(time.Hour).Unix(),
	}, audits: make(chan *poker.GtoDecisionRecord, 16)}
	db, err := gorm.Open(mysql.New(mysql.Config{
		DSN: "test@tcp(127.0.0.1:1)/test", SkipInitializeWithVersion: true,
	}), &gorm.Config{DisableAutomaticPing: true, SkipDefaultTransaction: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Callback().Query().Replace("gorm:query", func(tx *gorm.DB) {
		state.calls++
		where, ok := tx.Statement.Clauses["WHERE"].Expression.(clause.Where)
		if !ok || len(where.Exprs) != 1 {
			t.Fatal("activation query must filter by user_id")
		}
		expr, ok := where.Exprs[0].(clause.Expr)
		if !ok || expr.SQL != "user_id = ?" || len(expr.Vars) != 1 {
			t.Fatal("activation query must use a parameterized user_id")
		}
		state.users = append(state.users, expr.Vars[0].(string))
		if state.err != nil {
			tx.AddError(state.err)
			return
		}
		rows, ok := tx.Statement.Dest.(*[]*poker.UserActiveCode)
		if !ok {
			t.Errorf("unexpected query destination %T", tx.Statement.Dest)
			return
		}
		if state.info != nil {
			info := *state.info
			*rows = []*poker.UserActiveCode{&info}
		}
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.Callback().Create().Replace("gorm:create", func(tx *gorm.DB) {
		switch record := tx.Statement.Dest.(type) {
		case **poker.UserActiveCode:
			info := **record
			state.info = &info
			state.billings++
		case *poker.GtoDecisionRecord:
			state.audits <- record
		default:
			t.Errorf("unexpected create destination %T", tx.Statement.Dest)
		}
	}); err != nil {
		t.Fatal(err)
	}
	previous := dbmysql.DB
	dbmysql.DB = db
	t.Cleanup(func() {
		dbmysql.DB = previous
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	return state
}
