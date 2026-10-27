package ddbaby

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestPokerEventJSONPreservesWireTokens(t *testing.T) {
	for _, token := range []string{`{"large":9007199254740993}`, `[1,"two",null]`, `"encoded payload"`, `9007199254740993`, `null`, `true`} {
		t.Run(token, func(t *testing.T) {
			body := `{"packageName":"com.dpzx.pro","uid":9007199254740993,"data":` + token + `}`
			var req TexasPokerEventReq
			if err := json.Unmarshal([]byte(body), &req); err != nil {
				t.Fatal(err)
			}
			if string(req.Data) != token || string(req.UID) != `9007199254740993` {
				t.Fatalf("tokens changed: uid=%s data=%s", req.UID, req.Data)
			}
			encoded, err := json.Marshal(req)
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(encoded, &fields); err != nil {
				t.Fatal(err)
			}
			if string(fields["data"]) != token || string(fields["uid"]) != `9007199254740993` {
				t.Fatalf("wire tokens changed: %s", encoded)
			}
		})
	}
}

func TestTexasPokerEventResponsePreservesExistingWireShape(t *testing.T) {
	result, action := "Call 100", "call"
	amount, target := int64(0), int64(4975)
	base := &BaseResp{StatusMessage: "success", StatusCode: 0}
	tests := []struct {
		name     string
		response TexasPokerEventResp
		want     string
	}{
		{name: "acknowledgement", response: TexasPokerEventResp{BaseResp: base}, want: `{"BaseResp":{"StatusMessage":"success","StatusCode":0}}`},
		{name: "error", response: TexasPokerEventResp{BaseResp: &BaseResp{StatusCode: 503, StatusMessage: "decision_retry"}}, want: `{"BaseResp":{"StatusMessage":"decision_retry","StatusCode":503}}`},
		{name: "preflop decision", response: TexasPokerEventResp{BaseResp: base, Result: &result, HeroCard: []string{"10-H", "6-S"}, DecisionAction: &action, DecisionAmount: &amount, HeroMaxTarget: &target}, want: `{"BaseResp":{"StatusMessage":"success","StatusCode":0},"Result":"Call 100","HeroCard":["10-H","6-S"],"CommunityCard":null,"DecisionAction":"call","DecisionAmount":0,"HeroMaxTarget":4975}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			encoded, err := json.Marshal(test.response)
			if err != nil {
				t.Fatal(err)
			}
			var got, want any
			if err := json.Unmarshal(encoded, &got); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(test.want), &want); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("got %s, want %s", encoded, test.want)
			}
		})
	}
}
