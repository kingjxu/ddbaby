package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"regexp"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/sirupsen/logrus"
)

const maxTexasPokerEventBytes = 512 * 1024

var eventCommandPattern = regexp.MustCompile(`^[A-Za-z0-9._:/-]{1,128}$`)

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

	logrus.WithContext(ctx).WithFields(logrus.Fields{
		"cmd":          event.Cmd,
		"direction":    event.Direction,
		"location":     event.Location,
		"payload_size": len(event.Data),
		"event":        string(body),
	}).Info("[TexasPokerEvent] accepted")

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
