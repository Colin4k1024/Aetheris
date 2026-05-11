package http

import (
	"context"
	_ "embed"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

//go:embed static/homepage/index.html
var homepageHTML string

// HomePage 返回公开首页，帮助首次访问者理解 Aetheris 的价值与关键入口。
func (h *Handler) HomePage(ctx context.Context, c *app.RequestContext) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.SetStatusCode(consts.StatusOK)
	_, _ = c.WriteString(homepageHTML)
}
