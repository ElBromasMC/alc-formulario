package main

import (
	"alc/assets"
	"alc/config"
	"alc/handler"
	"alc/repository"
	"alc/service"
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"log"
	"net/http"
	"os"
	_ "time/tzdata"
)

func main() {
	// --- Load Config ---
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("could not load config: %v", err)
	}

	// --- DB connection setup ---
	connStr := os.Getenv("POSTGRESQL_URL")
	dbpool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer dbpool.Close()

	repo := repository.New(dbpool)

	e := echo.New()
	if os.Getenv("ENV") == "development" {
		e.Debug = true
	}

	// --- Middleware ---
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.RemoveTrailingSlashWithConfig(middleware.TrailingSlashConfig{
		RedirectCode: http.StatusMovedPermanently,
	}))
	e.Use(middleware.GzipWithConfig(middleware.GzipConfig{Level: 5}))

	// --- Services ---
	emailSvc, err := service.NewEmailService(cfg)
	if err != nil {
		log.Fatalf("could not create email service: %v", err)
	}
	certSvc := service.NewCertificateService(dbpool, repo, emailSvc)

	// --- Handlers ---
	authHandler := &handler.AuthHandler{Repo: repo}
	adminHandler := &handler.AdminHandler{Repo: repo, DBPool: dbpool}
	certHandler := &handler.CertificateHandler{Repo: repo, CertSvc: certSvc}
	apiHandler := &handler.ApiHandler{Repo: repo}
	dashboardHandler := &handler.DashboardHandler{Repo: repo}

	// Static files
	e.StaticFS("/static", echo.MustSubFS(assets.Assets, "static"))

	// Public routes
	e.GET("/login", authHandler.ShowLoginPage)
	e.POST("/login", authHandler.HandleLogin)
	e.GET("/logout", authHandler.HandleLogout)

	// Protected dashboard route
	dashboardGroup := e.Group("/dashboard")
	dashboardGroup.Use(handler.RequireAuth(repo))
	dashboardGroup.GET("", dashboardHandler.ShowDashboard)

	// Protected ADMIN routes
	adminGroup := e.Group("/admin")
	adminGroup.Use(handler.RequireAuth(repo), handler.RequireAdmin())
	adminGroup.GET("", adminHandler.ShowAdminDashboard)
	adminGroup.POST("/users", adminHandler.HandleCreateUser)

	adminGroup.POST("/software", adminHandler.HandleCreateSoftware)
	adminGroup.POST("/peripherals", adminHandler.HandleCreatePeripheral)
	adminGroup.POST("/config-items", adminHandler.HandleCreateConfigurationItem)
	adminGroup.POST("/upload/machine-users", adminHandler.HandleBulkUploadMachineUsers)
	adminGroup.POST("/upload/machines", adminHandler.HandleBulkUploadMachines)
	adminGroup.GET("/report/download", adminHandler.HandleDownloadReport)

	// Protected Certificate Routes
	certGroup := e.Group("/certificates")
	certGroup.Use(handler.RequireAuth(repo))
	certGroup.GET("/new", certHandler.ShowCertificateForm)
	certGroup.POST("/new", certHandler.HandleCreateCertificate)

	editGroup := e.Group("/certificate/edit")
	editGroup.Use(handler.RequireAuth(repo))
	editGroup.GET("/:id", certHandler.ShowEditCertificateForm)
	editGroup.POST("/:id", certHandler.HandleUpdateCertificate)

	apiGroup := e.Group("/api")
	apiGroup.Use(handler.RequireAuth(repo))
	apiGroup.GET("/machine-user", apiHandler.GetMachineUser)
	apiGroup.GET("/machine", apiHandler.GetMachine)

	// Public Confirmation Routes
	e.GET("/confirm/:token", certHandler.HandleCertificateConfirmation)
	e.GET("/reject/:token", certHandler.HandleCertificateRejection)
	e.GET("/certificate/view/:token", certHandler.ShowCertificate)

	// Add a root redirect for convenience
	e.GET("/", func(c echo.Context) error {
		return c.Redirect(http.StatusFound, "/dashboard")
	})

	// Start server
	log.Fatalln(e.Start(":8080"))
}
