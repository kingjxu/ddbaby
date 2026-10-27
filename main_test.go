package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
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
