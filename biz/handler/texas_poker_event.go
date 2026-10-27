package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"hash/fnv"
	"io"
	"net"
	"regexp"
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
	eventSourceBucketCount         = 256
	aapokerPackageName             = "com.plusaa.amula"
	aapokerAlternatePackageName    = "com.ty.aapk"
	dpzxPackageName                = "com.dpzx.pro"
)

var eventCommandPattern = regexp.MustCompile(`^[A-Za-z0-9._:/-]{1,128}$`)

var texasPokerEventIngress = newEventIngressGuard(
	defaultEventRequestsPerMinute,
	defaultSourceRequestsPerMinute,
)

var texasPokerEventStates = newAAPokerEventRegistry()
var executeAAPokerDecisionForEvent = executeAAPokerDecision

type eventIngressWindow struct {
	minute   int64
	requests int
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
	global                  eventIngressWindow
	sources                 [eventSourceBucketCount]eventSourceBucket
	dropped                 uint64
}

func newEventIngressGuard(globalRequests, sourceRequests int) *eventIngressGuard {
	return &eventIngressGuard{
		globalRequestsPerMinute: globalRequests,
		sourceRequestsPerMinute: sourceRequests,
	}
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

type texasPokerEvent struct {
	PackageName string          `json:"packageName"`
	UUID        string          `json:"uuid"`
	VPIP        int32           `json:"vpip"`
	PFR         int32           `json:"pfr"`
	Timestamp   int64           `json:"timestamp"`
	PID         json.RawMessage `json:"pid"`
	UID         json.RawMessage `json:"uid"`
	PacketID    json.RawMessage `json:"PacketId"`
	Cmd         string          `json:"cmd"`
	Direction   string          `json:"direction"`
	Data        json.RawMessage `json:"data"`
	Location    string          `json:"location"`
	Seq         json.RawMessage `json:"seq"`
	DataExtra   json.RawMessage `json:"dataExtra"`
}

// TexasPokerEvent receives one normalized poker-app event and dispatches it by
// the explicit package allowlist after validation and full diagnostic logging.
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
		"cmd":               event.Cmd,
		"direction":         event.Direction,
		"event":             string(body),
		"full_event_logged": true,
		"location":          event.Location,
		"package_name":      event.PackageName,
		"payload_size":      len(event.Data),
	}
	logrus.WithContext(ctx).WithFields(fields).Info("[TexasPokerEvent] accepted")

	response := map[string]any{
		"BaseResp": map[string]any{
			"StatusCode":    0,
			"StatusMessage": "success",
		},
	}
	if event.PackageName == dpzxPackageName {
		c.JSON(consts.StatusOK, response)
		return
	}
	candidate, stateErr := texasPokerEventStates.process(event, now)
	if stateErr != nil {
		logrus.WithContext(ctx).WithFields(fields).
			Warnf("[TexasPokerEvent] state update ignored:%v", stateErr)
	} else if candidate != nil {
		decision := candidate.Cached
		if decision == nil {
			token := texasPokerEventBearerToken(c)
			var decisionErr error
			decision, decisionErr = executeAAPokerDecisionForEvent(
				ctx, event.UUID, token, event.VPIP, event.PFR, candidate.Request,
			)
			if decisionErr != nil {
				texasPokerEventStates.complete(candidate, nil, now)
				logrus.WithContext(ctx).WithFields(fields).
					Warnf("[TexasPokerEvent] decision retry required:%v", decisionErr)
				writeTexasPokerEventError(c, consts.StatusServiceUnavailable, "decision_retry")
				return
			}
			texasPokerEventStates.complete(candidate, decision, now)
		}
		writeAAPokerDecisionResponse(response, decision)
	}
	c.JSON(consts.StatusOK, response)
}

func writeAAPokerDecisionResponse(response map[string]any, decision *aapokerDecisionResult) {
	if decision == nil {
		return
	}
	response["Result"] = decision.Result
	response["HeroCard"] = decision.HeroCard
	response["CommunityCard"] = decision.CommunityCard
	response["DecisionAction"] = decision.DecisionAction
	response["DecisionAmount"] = decision.DecisionAmount
	response["HeroMaxTarget"] = decision.HeroMaxTarget
}

func texasPokerEventBearerToken(c *app.RequestContext) string {
	header := strings.TrimSpace(string(c.Request.Header.Peek("Authorization")))
	if len(header) < 8 || !strings.EqualFold(header[:7], "Bearer ") {
		return ""
	}
	token := strings.TrimSpace(header[7:])
	if token == "" || strings.ContainsAny(token, " \t\r\n") || len(token) > 4096 {
		return ""
	}
	return token
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
	if event.PackageName != aapokerPackageName &&
		event.PackageName != aapokerAlternatePackageName &&
		event.PackageName != dpzxPackageName {
		return nil, errors.New("invalid packageName")
	}
	if len(event.UUID) > 128 || (event.UUID != "" && strings.TrimSpace(event.UUID) != event.UUID) ||
		!isSupportedDecisionProfile(event.VPIP, event.PFR) {
		return nil, errors.New("invalid decision identity or profile")
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
