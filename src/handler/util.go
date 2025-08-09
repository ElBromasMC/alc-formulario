package handler

import (
	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
	"net/http"
)

func render(ctx echo.Context, statusCode int, t templ.Component) error {
	buf := templ.GetBuffer()
	defer templ.ReleaseBuffer(buf)

	if err := t.Render(ctx.Request().Context(), buf); err != nil {
		return err
	}

	return ctx.HTML(statusCode, buf.String())
}

func renderOK(ctx echo.Context, t templ.Component) error {
	return render(ctx, http.StatusOK, t)
}
