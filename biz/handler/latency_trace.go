package handler

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/kingjxu/ddbaby/util"
	"github.com/sirupsen/logrus"
)

const (
	traceIDHeader              = "X-GTO-Trace-ID"
	clientStagesHeader         = "X-GTO-Client-Stages"
	previousClientTraceHeader  = "X-GTO-Previous-Trace"
	serverTimingHeader         = "X-GTO-Server-Timing"
	maxLatencyTraceHeaderBytes = 512
)

type decisionLatency struct {
	traceID           string
	requestReceivedAt time.Time
	handleStartedAt   time.Time
	authDoneAt        time.Time
	recognizeDoneAt   time.Time
	decisionStartedAt time.Time
	decisionDoneAt    time.Time
}

type clientStageTrace struct {
	TraceID           string `json:"trace_id"`
	DecisionCycleID   string `json:"decision_cycle_id,omitempty"`
	CycleStartFrame   bool   `json:"cycle_start_frame,omitempty"`
	HeroTurnToCapture int64  `json:"hero_turn_to_capture_ms,omitempty"`
	HeroTurnToHTTP    int64  `json:"hero_turn_to_http_start_ms,omitempty"`
	CaptureToReady    int64  `json:"capture_to_ready_ms"`
	QueueWait         int64  `json:"queue_wait_ms"`
	QueueBacklog      int64  `json:"queue_backlog_ms"`
	Superseded        int64  `json:"superseded_frames"`
	Compress          int64  `json:"compress_ms"`
	CaptureToHTTP     int64  `json:"capture_to_http_start_ms"`
}

type completedClientTrace struct {
	TraceID           string `json:"trace_id"`
	DecisionCycleID   string `json:"decision_cycle_id,omitempty"`
	FirstCycleDisplay bool   `json:"first_cycle_display,omitempty"`
	HeroTurnToDisplay int64  `json:"hero_turn_to_display_ms,omitempty"`
	CaptureToReady    int64  `json:"capture_to_ready_ms"`
	QueueWait         int64  `json:"queue_wait_ms"`
	QueueBacklog      int64  `json:"queue_backlog_ms"`
	Superseded        int64  `json:"superseded_frames"`
	Compress          int64  `json:"compress_ms"`
	RequestBodyWrite  int64  `json:"request_body_write_ms"`
	HTTP              int64  `json:"http_ms"`
	ResponseToDisplay int64  `json:"response_to_display_ms"`
	Total             int64  `json:"total_ms"`
	PendingTotal      int64  `json:"pending_total_ms"`
	ServerBind        int64  `json:"server_bind_ms"`
	ServerAuth        int64  `json:"server_auth_ms"`
	ServerRecognize   int64  `json:"server_recognize_ms"`
	ServerPrepare     int64  `json:"server_prepare_ms"`
	ServerDecision    int64  `json:"server_decision_ms"`
	ServerTotal       int64  `json:"server_total_ms"`
}

func validTraceID(value string) bool {
	if value == "" || len(value) > 64 {
		return false
	}
	for _, character := range value {
		if (character >= 'a' && character <= 'z') ||
			(character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') ||
			character == '-' || character == '_' || character == '.' || character == ':' {
			continue
		}
		return false
	}
	return true
}

func parseClientStageTrace(traceID, value string) (*clientStageTrace, bool) {
	if !validTraceID(traceID) || value == "" || len(value) > maxLatencyTraceHeaderBytes {
		return nil, false
	}
	parts := strings.Split(value, ",")
	version := parts[0]
	numberStart := 1
	cycleID := ""
	var heroTurnDetected int64
	cycleStartFrame := false
	if version == "v2" {
		if len(parts) != 13 || !validTraceID(parts[1]) {
			return nil, false
		}
		cycleID = parts[1]
		detected, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil || detected < 0 || (parts[3] != "0" && parts[3] != "1") {
			return nil, false
		}
		heroTurnDetected = detected
		cycleStartFrame = parts[3] == "1"
		numberStart = 4
	} else if version != "v1" || len(parts) != 10 {
		return nil, false
	}
	numbers := make([]int64, 0, 9)
	for _, part := range parts[numberStart:] {
		number, err := strconv.ParseInt(part, 10, 64)
		if err != nil || number < 0 {
			return nil, false
		}
		numbers = append(numbers, number)
	}
	capture, ready, enqueue, pendingSince, dequeue, superseded, compressStart, compressEnd, httpStart :=
		numbers[0], numbers[1], numbers[2], numbers[3], numbers[4], numbers[5], numbers[6], numbers[7], numbers[8]
	if !(pendingSince <= capture && capture <= ready && ready <= enqueue && enqueue <= dequeue &&
		dequeue <= compressStart && superseded <= 1000 &&
		compressStart <= compressEnd && compressEnd <= httpStart && httpStart-capture <= 60000) {
		return nil, false
	}
	trace := &clientStageTrace{
		TraceID: traceID, CaptureToReady: ready - capture, QueueWait: dequeue - enqueue,
		QueueBacklog: dequeue - pendingSince, Superseded: superseded,
		Compress: compressEnd - compressStart, CaptureToHTTP: httpStart - capture,
	}
	if cycleID != "" {
		if heroTurnDetected > capture || capture-heroTurnDetected > 60000 {
			return nil, false
		}
		trace.DecisionCycleID = cycleID
		trace.CycleStartFrame = cycleStartFrame
		trace.HeroTurnToCapture = capture - heroTurnDetected
		trace.HeroTurnToHTTP = httpStart - heroTurnDetected
	}
	return trace, true
}

func parseCompletedClientTrace(value string) (*completedClientTrace, bool) {
	parts := strings.Split(value, ",")
	if len(parts) == 0 {
		return nil, false
	}
	version := parts[0]
	expected := 21
	numberStart := 2
	cycleID := ""
	firstCycleDisplay := false
	var heroTurnDetected int64
	if version == "v2" {
		expected = 24
		numberStart = 5
	} else if version != "v1" {
		return nil, false
	}
	if len(parts) != expected || len(value) > maxLatencyTraceHeaderBytes ||
		!validTraceID(parts[1]) {
		return nil, false
	}
	if version == "v2" {
		if !validTraceID(parts[2]) || (parts[4] != "0" && parts[4] != "1") {
			return nil, false
		}
		detected, err := strconv.ParseInt(parts[3], 10, 64)
		if err != nil || detected < 0 {
			return nil, false
		}
		cycleID = parts[2]
		heroTurnDetected = detected
		firstCycleDisplay = parts[4] == "1"
	}
	numbers := make([]int64, 0, 19)
	for _, part := range parts[numberStart:] {
		number, err := strconv.ParseInt(part, 10, 64)
		if err != nil || number < 0 {
			return nil, false
		}
		numbers = append(numbers, number)
	}
	if len(numbers) != 19 {
		return nil, false
	}
	capture, ready, enqueue, pendingSince, dequeue, superseded :=
		numbers[0], numbers[1], numbers[2], numbers[3], numbers[4], numbers[5]
	compressStart, compressEnd := numbers[6], numbers[7]
	bodyStart, bodyEnd, httpStart := numbers[8], numbers[9], numbers[10]
	responseEnd, displayEnd := numbers[11], numbers[12]
	serverTiming := numbers[13:]
	if !(pendingSince <= capture && capture <= ready && ready <= enqueue && enqueue <= dequeue &&
		dequeue <= compressStart && superseded <= 1000 &&
		compressStart <= compressEnd && compressEnd <= httpStart && httpStart <= bodyStart && bodyStart <= bodyEnd &&
		bodyEnd <= responseEnd && responseEnd <= displayEnd && displayEnd-pendingSince <= 60000) {
		return nil, false
	}
	for _, duration := range serverTiming {
		if duration > 60000 {
			return nil, false
		}
	}
	trace := &completedClientTrace{
		TraceID: parts[1], CaptureToReady: ready - capture, QueueWait: dequeue - enqueue,
		QueueBacklog: dequeue - pendingSince, Superseded: superseded,
		Compress: compressEnd - compressStart, RequestBodyWrite: bodyEnd - bodyStart,
		HTTP: responseEnd - httpStart, ResponseToDisplay: displayEnd - responseEnd,
		Total: displayEnd - capture, PendingTotal: displayEnd - pendingSince,
		ServerBind: numbers[13], ServerAuth: numbers[14],
		ServerRecognize: numbers[15], ServerPrepare: numbers[16], ServerDecision: numbers[17],
		ServerTotal: numbers[18],
	}
	if cycleID != "" {
		if heroTurnDetected > capture || displayEnd-heroTurnDetected > 60000 {
			return nil, false
		}
		trace.DecisionCycleID = cycleID
		trace.FirstCycleDisplay = firstCycleDisplay
		trace.HeroTurnToDisplay = displayEnd - heroTurnDetected
	}
	return trace, true
}

func elapsedOrMissing(start, end time.Time) int64 {
	if start.IsZero() || end.IsZero() || end.Before(start) {
		return -1
	}
	return end.Sub(start).Milliseconds()
}

func (latency *decisionLatency) writeResponseHeader(ctx *app.RequestContext) {
	if latency.traceID == "" {
		return
	}
	now := time.Now()
	bind := elapsedOrMissing(latency.requestReceivedAt, latency.handleStartedAt)
	auth := elapsedOrMissing(latency.handleStartedAt, latency.authDoneAt)
	recognize := elapsedOrMissing(latency.authDoneAt, latency.recognizeDoneAt)
	prepare := elapsedOrMissing(latency.recognizeDoneAt, latency.decisionStartedAt)
	decision := elapsedOrMissing(latency.decisionStartedAt, latency.decisionDoneAt)
	total := elapsedOrMissing(latency.requestReceivedAt, now)
	ctx.Header(traceIDHeader, latency.traceID)
	ctx.Header(serverTimingHeader, fmt.Sprintf("v1,%d,%d,%d,%d,%d,%d",
		bind, auth, recognize, prepare, decision, total))
	logrus.Infof("[GtoLatencyTrace] trace_id=%s server_timing=%s",
		latency.traceID, ctx.Response.Header.Get(serverTimingHeader))
}

func logClientTraceHeaders(ctx *app.RequestContext, traceID string) {
	if current, ok := parseClientStageTrace(traceID, string(ctx.Request.Header.Peek(clientStagesHeader))); ok {
		logrus.Infof("[GtoLatencyTrace] client_stage:%s", util.ToJSON(current))
	}
	previous := string(ctx.Request.Header.Peek(previousClientTraceHeader))
	if previous == "" {
		return
	}
	if completed, ok := parseCompletedClientTrace(previous); ok {
		if completed.DecisionCycleID != "" && !completed.FirstCycleDisplay {
			logrus.Infof("[GtoLatencyTrace] client_complete_repeat:%s", util.ToJSON(completed))
		} else {
			logrus.Infof("[GtoLatencyTrace] client_complete:%s", util.ToJSON(completed))
		}
	} else {
		logrus.Warn("[GtoLatencyTrace] invalid previous client trace header")
	}
}
