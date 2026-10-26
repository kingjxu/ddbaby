package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"hash/fnv"
	"io"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/sirupsen/logrus"
)

const (
	maxTexasPokerEventBytes        = 512 * 1024
	defaultEventRequestsPerMinute  = 300
	defaultSourceRequestsPerMinute = 120
	defaultEventLogBytesPerMinute  = 6 * 1024 * 1024
	eventSourceBucketCount         = 256
	eventLogRecordOverheadBytes    = 1024
	maxEventFullLogDuration        = 24 * time.Hour
)

var eventCommandPattern = regexp.MustCompile(`^[A-Za-z0-9._:/-]{1,128}$`)

var texasPokerEventIngress = newEventIngressGuard(
	defaultEventRequestsPerMinute,
	defaultSourceRequestsPerMinute,
	defaultEventLogBytesPerMinute,
	fullEventLoggingUntil(time.Now()),
)

type eventIngressWindow struct {
	minute   int64
	requests int
	logBytes int
}

type eventSourceBucket struct {
	minute   int64
	key      uint64
	requests int
}

// eventIngressGuard uses a fixed number of source buckets so spoofed source
// addresses cannot grow server memory. Hash collisions share a quota, which is
// conservative and cannot increase the global allowance.
type eventIngressGuard struct {
	mu                      sync.Mutex
	globalRequestsPerMinute int
	sourceRequestsPerMinute int
	logBytesPerMinute       int
	fullLogUntil            time.Time
	global                  eventIngressWindow
	sources                 [eventSourceBucketCount]eventSourceBucket
	dropped                 uint64
}

func newEventIngressGuard(globalRequests, sourceRequests, logBytes int, fullLogUntil time.Time) *eventIngressGuard {
	return &eventIngressGuard{
		globalRequestsPerMinute: globalRequests,
		sourceRequestsPerMinute: sourceRequests,
		logBytesPerMinute:       logBytes,
		fullLogUntil:            fullLogUntil,
	}
}

func fullEventLoggingUntil(now time.Time) time.Time {
	value := strings.ToLower(strings.TrimSpace(os.Getenv("DDBABY_GTO_EVENT_FULL_LOG")))
	if value != "1" && value != "true" {
		return time.Time{}
	}
	deadlineSeconds, err := strconv.ParseInt(strings.TrimSpace(os.Getenv("DDBABY_GTO_EVENT_FULL_LOG_UNTIL")), 10, 64)
	if err != nil {
		return time.Time{}
	}
	deadline := time.Unix(deadlineSeconds, 0)
	if !deadline.After(now) || deadline.After(now.Add(maxEventFullLogDuration)) {
		return time.Time{}
	}
	return deadline
}

func eventSourceKey(source string) uint64 {
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(source))
	return hasher.Sum64()
}

func texasPokerEventSource(c *app.RequestContext) string {
	remote := c.RemoteAddr().String()
	host, _, err := net.SplitHostPort(remote)
	if err == nil {
		remote = host
	}
	remoteIP := net.ParseIP(remote)
	if remoteIP != nil && remoteIP.IsLoopback() {
		// Forwarded headers are trusted only from the local reverse proxy.
		return c.ClientIP()
	}
	return remote
}

func (g *eventIngressGuard) allowRequest(now time.Time, source string) (bool, uint64) {
	minute := now.Unix() / 60
	key := eventSourceKey(source)
	index := key % eventSourceBucketCount

	g.mu.Lock()
	defer g.mu.Unlock()

	var droppedToReport uint64
	if g.global.minute != minute {
		droppedToReport = g.dropped
		g.dropped = 0
		g.global = eventIngressWindow{minute: minute}
	}
	bucket := &g.sources[index]
	if bucket.minute != minute {
		*bucket = eventSourceBucket{minute: minute, key: key}
	}
	// A same-minute hash collision shares the bucket's current allowance
	// rather than replacing it and letting either source bypass its quota.

	if g.global.requests >= g.globalRequestsPerMinute || bucket.requests >= g.sourceRequestsPerMinute {
		g.dropped++
		return false, droppedToReport
	}
	g.global.requests++
	bucket.requests++
	return true, droppedToReport
}

func (g *eventIngressGuard) allowFullLog(now time.Time, bytes int) bool {
	if g.fullLogUntil.IsZero() || !now.Before(g.fullLogUntil) || bytes <= 0 {
		return false
	}
	minute := now.Unix() / 60
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.global.minute != minute {
		g.global = eventIngressWindow{minute: minute}
		g.dropped = 0
	}
	if bytes > g.logBytesPerMinute-g.global.logBytes {
		return false
	}
	g.global.logBytes += bytes
	return true
}

func eventFullLogCost(bodyBytes int) int {
	// Logrus quotes string fields, so quotes and backslashes can expand the
	// serialized record. Charge twice the body plus fixed metadata overhead.
	return bodyBytes*2 + eventLogRecordOverheadBytes
}

type texasPokerEvent struct {
	Timestamp int64           `json:"timestamp"`
	PID       json.RawMessage `json:"pid"`
	UID       json.RawMessage `json:"uid"`
	PacketID  json.RawMessage `json:"PacketId"`
	Cmd       string          `json:"cmd"`
	Direction string          `json:"direction"`
	Data      json.RawMessage `json:"data"`
	Location  string          `json:"location"`
	Seq       json.RawMessage `json:"seq"`
	DataExtra json.RawMessage `json:"dataExtra"`
}

// TexasPokerEvent receives one normalized AAPoker protocol event. Version one
// intentionally acknowledges receipt only; state reconstruction is connected
// behind this boundary in a later change.
func TexasPokerEvent(ctx context.Context, c *app.RequestContext) {
	texasPokerEventWithIngress(ctx, c, texasPokerEventIngress, time.Now())
}

func texasPokerEventWithIngress(ctx context.Context, c *app.RequestContext, ingress *eventIngressGuard, now time.Time) {
	allowed, droppedToReport := ingress.allowRequest(now, texasPokerEventSource(c))
	if droppedToReport > 0 {
		logrus.WithContext(ctx).WithField("dropped_count", droppedToReport).
			Warn("[TexasPokerEvent] rate limited requests")
	}
	if !allowed {
		writeTexasPokerEventError(c, consts.StatusTooManyRequests, "rate_limited")
		return
	}

	body, err := readTexasPokerEventBody(c)
	if err != nil {
		writeTexasPokerEventError(c, consts.StatusBadRequest, "invalid_body_size")
		return
	}

	event, err := parseTexasPokerEvent(body)
	if err != nil {
		writeTexasPokerEventError(c, consts.StatusBadRequest, "invalid_event")
		return
	}

	fields := logrus.Fields{
		"cmd":          event.Cmd,
		"direction":    event.Direction,
		"location":     event.Location,
		"payload_size": len(event.Data),
	}
	if ingress.allowFullLog(now, eventFullLogCost(len(body))) {
		fields["event"] = string(body)
		fields["full_event_logged"] = true
	} else {
		fields["full_event_logged"] = false
	}
	logrus.WithContext(ctx).WithFields(fields).Info("[TexasPokerEvent] accepted")

	c.JSON(consts.StatusOK, map[string]any{
		"BaseResp": map[string]any{
			"StatusCode":    0,
			"StatusMessage": "success",
		},
	})
}

func readTexasPokerEventBody(c *app.RequestContext) ([]byte, error) {
	if !c.Request.IsBodyStream() {
		body := c.Request.Body()
		if len(body) == 0 || len(body) > maxTexasPokerEventBytes {
			return nil, errors.New("invalid body size")
		}
		return body, nil
	}

	contentLength := c.Request.Header.ContentLength()
	if contentLength == 0 || contentLength > maxTexasPokerEventBytes {
		return nil, errors.New("invalid body size")
	}

	body, err := io.ReadAll(io.LimitReader(
		c.RequestBodyStream(), int64(maxTexasPokerEventBytes)+1,
	))
	if err != nil || len(body) == 0 || len(body) > maxTexasPokerEventBytes {
		return nil, errors.New("invalid body size")
	}
	return body, nil
}

func parseTexasPokerEvent(body []byte) (*texasPokerEvent, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	var event texasPokerEvent
	if err := decoder.Decode(&event); err != nil {
		return nil, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, errors.New("multiple JSON values")
	}
	if event.Timestamp <= 0 || !eventCommandPattern.MatchString(event.Cmd) {
		return nil, errors.New("invalid timestamp or command")
	}
	if event.Direction != "ServerToClient" && event.Direction != "ClientToServer" {
		return nil, errors.New("invalid direction")
	}
	if event.Location != "TABLE" && event.Location != "LOBBY" && event.Location != "UNKNOWN" {
		return nil, errors.New("invalid location")
	}
	if !validOptionalScalar(event.PID, 128) || !validOptionalScalar(event.UID, 128) ||
		!validOptionalScalar(event.DataExtra, 128) {
		return nil, errors.New("invalid optional metadata")
	}
	if len(event.Data) == 0 || len(event.Data) > 480*1024 || !json.Valid(event.Data) {
		return nil, errors.New("invalid data")
	}
	return &event, nil
}

func validOptionalScalar(raw json.RawMessage, maxBytes int) bool {
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return true
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return false
	}
	switch typed := value.(type) {
	case string:
		return len([]byte(typed)) <= maxBytes
	case json.Number:
		return len(typed.String()) <= maxBytes
	default:
		return false
	}
}

func writeTexasPokerEventError(c *app.RequestContext, status int, code string) {
	c.JSON(status, map[string]any{
		"BaseResp": map[string]any{
			"StatusCode":    status,
			"StatusMessage": code,
		},
	})
}
