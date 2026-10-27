package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/sirupsen/logrus"
)

func TestServerStreamsRequestBodiesForRouteSpecificLimits(t *testing.T) {
	h := newHertzServer()
	if !h.IsStreamRequestBody() {
		t.Fatal("request body streaming must remain enabled")
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

func TestExistingRoutesRetainFourMiBBodyLimit(t *testing.T) {
	var requestContext app.RequestContext
	requestContext.Request.SetRequestURI("/existing")
	requestContext.Request.SetBodyStream(
		bytes.NewReader(make([]byte, defaultMaxRequestBodyBytes+1)), -1,
	)

	preserveDefaultRequestBodyLimit(context.Background(), &requestContext)
	if requestContext.Response.StatusCode() != consts.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", requestContext.Response.StatusCode())
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
