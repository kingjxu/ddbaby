package handler

import (
	"context"
	"encoding/json"
	"mime"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	ddbaby "github.com/kingjxu/ddbaby/biz/model/ddbaby"
)

// PrepareTexasPokerEvent enforces this route's HTTP envelope before binding.
func PrepareTexasPokerEvent(ctx context.Context, c *app.RequestContext) {
	body := c.Request.Body()
	mediaType, _, err := mime.ParseMediaType(string(c.Request.Header.ContentType()))
	var fields map[string]json.RawMessage
	if err != nil || mediaType != "application/json" || json.Unmarshal(body, &fields) != nil || fields == nil {
		writeTexasPokerEventError(c, consts.StatusBadRequest, "invalid_event")
		return
	}
	// Routing requires the exact protocol key. The generic JSON binder also
	// accepts case-insensitive aliases, which must not select another source.
	for name := range fields {
		if name != "packageName" && strings.EqualFold(name, "packageName") {
			writeTexasPokerEventError(c, consts.StatusBadRequest, "invalid_event")
			return
		}
	}
	var packageName string
	if json.Unmarshal(fields["packageName"], &packageName) != nil || packageName == "" {
		writeTexasPokerEventError(c, consts.StatusBadRequest, "invalid_event")
		return
	}
	// Generated models also have query tags; preserve the URI for access logs.
	query := string(c.Request.URI().QueryString())
	c.Request.URI().SetQueryString("")
	defer c.Request.URI().SetQueryString(query)
	c.Next(ctx)
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

func writeTexasPokerEventError(c *app.RequestContext, status int, code string) {
	c.Abort()
	c.JSON(status, &ddbaby.TexasPokerEventResp{
		BaseResp: &ddbaby.BaseResp{StatusCode: int32(status), StatusMessage: code},
	})
}
