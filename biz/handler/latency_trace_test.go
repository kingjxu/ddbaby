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

func TestParseClientStageTraceV2DecisionCycle(t *testing.T) {
	trace, ok := parseClientStageTrace(
		"1700000000000-12",
		"v2,cycle-1,80,1,100,112,115,70,140,2,141,151,153",
	)
	if !ok {
		t.Fatal("expected valid v2 trace")
	}
	if trace.DecisionCycleID != "cycle-1" || !trace.CycleStartFrame ||
		trace.HeroTurnToCapture != 20 || trace.HeroTurnToHTTP != 73 ||
		trace.QueueBacklog != 70 {
		t.Fatalf("unexpected v2 trace: %+v", trace)
	}

	invalid := []string{
		"v2,bad cycle,80,1,100,112,115,70,140,2,141,151,153",
		"v2,cycle-1,101,1,100,112,115,70,140,2,141,151,153",
		"v2,cycle-1,80,2,100,112,115,70,140,2,141,151,153",
		"v2,cycle-1,80,1,100,112,115,70,140,2,141,151",
	}
	for _, value := range invalid {
		if _, accepted := parseClientStageTrace("trace", value); accepted {
			t.Fatalf("accepted invalid v2 trace: %q", value)
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

func TestParseCompletedClientTraceV2DecisionCycle(t *testing.T) {
	value := "v2,trace-1,cycle-1,80,1,100,112,115,70,140,2,141,151,154,160,153,300,306,2,8,120,10,45,166"
	trace, ok := parseCompletedClientTrace(value)
	if !ok {
		t.Fatal("expected valid completed v2 trace")
	}
	if trace.DecisionCycleID != "cycle-1" || !trace.FirstCycleDisplay ||
		trace.HeroTurnToDisplay != 226 || trace.PendingTotal != 236 ||
		trace.ServerRecognize != 120 {
		t.Fatalf("unexpected completed v2 trace: %+v", trace)
	}

	invalid := []string{
		"v2,trace-1,bad cycle,80,1,100,112,115,70,140,2,141,151,154,160,153,300,306,2,8,120,10,45,166",
		"v2,trace-1,cycle-1,101,1,100,112,115,70,140,2,141,151,154,160,153,300,306,2,8,120,10,45,166",
		"v2,trace-1,cycle-1,80,2,100,112,115,70,140,2,141,151,154,160,153,300,306,2,8,120,10,45,166",
		"v2,trace-1,cycle-1,80,1,100,112,115,70,140,2,141,151,154,160,153,300,306,2,8,120,10,45",
	}
	for _, item := range invalid {
		if _, accepted := parseCompletedClientTrace(item); accepted {
			t.Fatalf("accepted invalid completed v2 trace: %q", item)
		}
	}
}
