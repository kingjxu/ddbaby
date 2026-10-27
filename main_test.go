package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/test/mock"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/sirupsen/logrus"
)

func TestHertzAppliesUnifiedBufferedBodyLimit(t *testing.T) {
	h := newHertzServer()
	register(h)
	h.POST("/existing", func(ctx context.Context, c *app.RequestContext) {
		c.String(consts.StatusOK, "ok")
	})
	if err := h.Init(); err != nil {
		t.Fatal(err)
	}
	previous := logrus.StandardLogger().Out
	logrus.SetOutput(io.Discard)
	t.Cleanup(func() { logrus.SetOutput(previous) })
	for _, path := range []string{"/existing", "/lyxz/gto_event"} {
		for _, size := range []int{512*1024 + 1, defaultMaxRequestBodyBytes, defaultMaxRequestBodyBytes + 1} {
			for _, chunked := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%d/chunked=%t", path, size, chunked), func(t *testing.T) {
					prefix, suffix := `{"packageName":"com.dpzx.pro","data":"`, `"}`
					body := prefix + strings.Repeat("a", size-len(prefix)-len(suffix)) + suffix
					header := fmt.Sprintf("Content-Length: %d\r\n", len(body))
					if chunked {
						header = "Transfer-Encoding: chunked\r\n"
						body = fmt.Sprintf("%x\r\n%s\r\n0\r\n\r\n", len(body), body)
					}
					conn := mock.NewConn("POST " + path + " HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\nContent-Type: application/json\r\n" + header + "\r\n" + body)
					conn.SetReadTimeout(time.Second)
					_ = h.Serve(context.Background(), conn)
					recorder := conn.WriterRecorder()
					response, err := recorder.Peek(recorder.WroteLen())
					if err != nil {
						t.Fatal(err)
					}
					wantStatus := consts.StatusOK
					if size > defaultMaxRequestBodyBytes {
						wantStatus = consts.StatusRequestEntityTooLarge
					}
					if !strings.HasPrefix(string(response), fmt.Sprintf("HTTP/1.1 %d ", wantStatus)) {
						t.Fatalf("HTTP response = %.200s", response)
					}
				})
			}
		}
	}
}

func TestGeneratedEventRouteRetainsWireProtocol(t *testing.T) {
	h := newHertzServer()
	register(h) // Must not register the same route twice.
	for _, test := range []struct {
		body   string
		status int
	}{
		{body: `{"packageName":"com.dpzx.pro","data":[1,2,3]}`, status: 200},
		{body: `{"packageName":"unknown"}`, status: 400},
		{body: `{"packageName":"com.dpzx.pro","timestamp":"invalid"}`, status: 400},
	} {
		response := ut.PerformRequest(h.Engine, "POST", "/lyxz/gto_event", &ut.Body{Body: strings.NewReader(test.body), Len: len(test.body)},
			ut.Header{Key: "Content-Type", Value: "application/json"})
		if response.Code != test.status {
			t.Fatalf("response = %d %s", response.Code, response.Body.String())
		}
		if strings.Contains(response.Body.String(), "HeroCard") {
			t.Fatalf("basic response contains decision fields: %s", response.Body.String())
		}
	}
}

func TestEventRouteBindsOnlySingleJSONObjectBody(t *testing.T) {
	h := newHertzServer()
	register(h)
	for _, test := range []struct {
		name, contentType, body string
	}{
		{"plain text", "text/plain", `not-json-marker`},
		{"plain object", "text/plain", `{"packageName":"com.dpzx.pro","data":"marker"}`},
		{"missing content type", "", `{"packageName":"com.dpzx.pro","data":"marker"}`},
		{"missing package", "application/json", `{"data":"marker"}`},
		{"unknown package", "application/json", `{"packageName":"unknown","data":"marker"}`},
		{"invalid package type", "application/json", `{"packageName":123,"data":"marker"}`},
		{"uppercase alias only", "application/json", `{"PackageName":"com.dpzx.pro","data":"marker"}`},
		{"lowercase alias only", "application/json", `{"packagename":"com.dpzx.pro","data":"marker"}`},
		{"mixed case alias only", "application/json", `{"PACKAGEname":"com.dpzx.pro","data":"marker"}`},
		{"unknown exact before alias", "application/json", `{"packageName":"unknown","PackageName":"com.dpzx.pro","data":"marker"}`},
		{"unknown exact after alias", "application/json", `{"PackageName":"com.dpzx.pro","packageName":"unknown","data":"marker"}`},
		{"registered exact before alias", "application/json", `{"packageName":"com.dpzx.pro","PackageName":"com.plusaa.amula","data":"marker"}`},
		{"registered exact after alias", "application/json", `{"PackageName":"com.plusaa.amula","packageName":"com.dpzx.pro","data":"marker"}`},
		{"same value alias", "application/json", `{"packageName":"com.dpzx.pro","PackageName":"com.dpzx.pro","data":"marker"}`},
		{"invalid json", "application/json", `{"data":"marker"`},
		{"multiple values", "application/json", `{"packageName":"com.dpzx.pro","data":"marker"} {}`},
		{"array", "application/json", `[{"packageName":"com.dpzx.pro","data":"marker"}]`},
		{"null", "application/json", `null`},
	} {
		t.Run(test.name, func(t *testing.T) {
			var logs bytes.Buffer
			previous := logrus.StandardLogger().Out
			logrus.SetOutput(&logs)
			t.Cleanup(func() { logrus.SetOutput(previous) })
			response := ut.PerformRequest(h.Engine, "POST", "/lyxz/gto_event?PackageName=com.dpzx.pro",
				&ut.Body{Body: strings.NewReader(test.body), Len: len(test.body)},
				ut.Header{Key: "Content-Type", Value: test.contentType})
			if response.Code != consts.StatusBadRequest || strings.Contains(response.Body.String(), "marker") ||
				strings.Contains(logs.String(), "marker") || strings.Contains(logs.String(), "full_event_logged=true") {
				t.Fatalf("response/log = %d %s / %s", response.Code, response.Body.String(), logs.String())
			}
		})
	}

	// A conflicting query cannot change a registered body source or other fields.
	var logs bytes.Buffer
	previous := logrus.StandardLogger().Out
	logrus.SetOutput(&logs)
	t.Cleanup(func() { logrus.SetOutput(previous) })
	body := `{"packageName":"com.dpzx.pro","data":"body-marker"}`
	response := ut.PerformRequest(h.Engine, "POST", "/lyxz/gto_event?PackageName=com.plusaa.amula&Cmd=query-marker",
		&ut.Body{Body: strings.NewReader(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json; charset=utf-8"})
	if response.Code != consts.StatusOK || !strings.Contains(logs.String(), "package_name=com.dpzx.pro") ||
		strings.Contains(logs.String(), "query-marker") {
		t.Fatalf("query changed event: %d %s / %s", response.Code, response.Body.String(), logs.String())
	}
}
