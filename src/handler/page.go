package handler

import (
	"alc/view"
	"github.com/labstack/echo/v4"
)

func (h *Handler) HandleIndexShow(c echo.Context) error {
	return renderOK(c, view.Index())
}
