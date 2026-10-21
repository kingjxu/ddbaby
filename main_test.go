package main

import (
	"bytes"
	"context"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

func TestServerStreamsRequestBodiesForRouteSpecificLimits(t *testing.T) {
	h := newHertzServer()
	if !h.IsStreamRequestBody() {
		t.Fatal("request body streaming must remain enabled")
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
