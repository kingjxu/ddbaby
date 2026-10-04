package handler

import "testing"

func TestParseClientStageTrace(t *testing.T) {
	trace, ok := parseClientStageTrace("1700000000000-12", "v1,100,112,115,90,140,2,141,151,153")
	if !ok {
		t.Fatal("expected valid trace")
	}
	if trace.CaptureToReady != 12 || trace.QueueWait != 25 || trace.QueueBacklog != 50 ||
		trace.Superseded != 2 || trace.Compress != 10 || trace.CaptureToHTTP != 53 {
		t.Fatalf("unexpected trace: %+v", trace)
	}

	invalid := []struct{ id, value string }{
		{"bad\ntrace", "v1,100,112,115,90,140,2,141,151,153"},
		{"ok", "v1,100,90,115,90,140,2,141,151,153"},
		{"ok", "v1,100,112,115,90,140,2,141,151"},
		{"ok", "v1,100,112,115,90,140,2,141,151,70001"},
	}
	for _, item := range invalid {
		if _, accepted := parseClientStageTrace(item.id, item.value); accepted {
			t.Fatalf("accepted invalid trace id=%q value=%q", item.id, item.value)
		}
	}
}

func TestParseCompletedClientTrace(t *testing.T) {
	value := "v1,1700000000000-12,100,112,115,90,140,2,141,151,154,160,153,300,306,2,8,120,10,45,166"
	trace, ok := parseCompletedClientTrace(value)
	if !ok {
		t.Fatal("expected valid completed trace")
	}
	if trace.Total != 206 || trace.PendingTotal != 216 || trace.QueueBacklog != 50 ||
		trace.Superseded != 2 || trace.HTTP != 147 || trace.ResponseToDisplay != 6 ||
		trace.ServerRecognize != 120 || trace.ServerTotal != 166 {
		t.Fatalf("unexpected completed trace: %+v", trace)
	}

	if _, ok := parseCompletedClientTrace(value + ",1"); ok {
		t.Fatal("accepted trace with extra field")
	}
	if _, ok := parseCompletedClientTrace("v1,bad id,100,112,115,140,141,151,154,160,153,300,306,2,8,120,10,45,166"); ok {
		t.Fatal("accepted invalid trace id")
	}
	if _, ok := parseCompletedClientTrace("v1,trace,100,112,115,90,140,2,141,151,154,160,153,300,306,2,8,-1,10,45,166"); ok {
		t.Fatal("accepted negative server timing")
	}
	if _, ok := parseCompletedClientTrace("v1,trace,100,112,115,90,140,2,141,151,154,160,153,300,306,2,8,60001,10,45,166"); ok {
		t.Fatal("accepted server timing above 60 seconds")
	}
	if _, ok := parseCompletedClientTrace("v1,trace,100,112,115,90,140,2,141,151,154,160,153,300,306,60000,60000,60000,60000,60000,60000"); !ok {
		t.Fatal("rejected valid 60 second server timing boundary")
	}
}
