package main

import (
	"alc/assets"
	"alc/handler"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"log"
	"net/http"
	"os"
)

func main() {
	e := echo.New()
	if os.Getenv("ENV") == "development" {
		e.Debug = true
	}

	// Initialize handlers
	h := handler.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.RemoveTrailingSlashWithConfig(middleware.TrailingSlashConfig{
		RedirectCode: http.StatusMovedPermanently,
	}))
	e.Use(middleware.GzipWithConfig(middleware.GzipConfig{Level: 5}))

	// Static files
	e.StaticFS("/static", echo.MustSubFS(assets.Assets, "static"))

	// Page routes
	e.GET("/", h.HandleIndexShow)

	// Start server
	log.Fatalln(e.Start(":8080"))
}
