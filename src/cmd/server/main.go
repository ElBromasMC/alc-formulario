package main

import (
	"alc/assets"
	"alc/handler"
	"alc/service"
	"context"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"log"
	"net/http"
	"os"

	"alc/model"
	"alc/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
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
	certSvc := service.NewCertificateService(dbpool, repo)

	// --- Handlers ---
	authHandler := &handler.AuthHandler{Repo: repo}
	adminHandler := &handler.AdminHandler{Repo: repo, DBPool: dbpool}
	certHandler := &handler.CertificateHandler{Repo: repo, CertSvc: certSvc}
	apiHandler := &handler.ApiHandler{Repo: repo}

	// Static files
	e.StaticFS("/static", echo.MustSubFS(assets.Assets, "static"))

	// Public routes
	e.GET("/login", authHandler.ShowLoginPage)
	e.POST("/login", authHandler.HandleLogin)
	e.GET("/logout", authHandler.HandleLogout)

	// A simple dashboard for non-admin users
	techDashboard := func(c echo.Context) error {
		// In a real app, this would be a proper handler and view.
		// It could link to the certificate form.
		html := `
			<div style="font-family: sans-serif; padding: 2rem;">
				<h1>Technician Dashboard</h1>
				<p>Welcome! You can now create a new certificate.</p>
				<a href="/certificates/new" style="display: inline-block; padding: 10px 15px; background-color: #003366; color: white; text-decoration: none; border-radius: 5px;">Create New Certificate</a>
				<br/><br/>
				<a href="/logout">Logout</a>
			</div>
		`
		return c.HTML(http.StatusOK, html)
	}

	// Protected dashboard route (for all logged-in users)
	dashboardGroup := e.Group("/dashboard")
	dashboardGroup.Use(handler.RequireAuth(repo))
	dashboardGroup.GET("", func(c echo.Context) error {
		user := c.Get("user").(model.AuthenticatedUser)
		if user.Role == repository.UserRoleADMIN {
			return c.Redirect(http.StatusFound, "/admin")
		}
		// Redirect to the tech dashboard
		return techDashboard(c)
	})

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

	apiGroup := e.Group("/api")
	apiGroup.Use(handler.RequireAuth(repo))
	apiGroup.GET("/machine-user", apiHandler.GetMachineUser)
	apiGroup.GET("/machine", apiHandler.GetMachine)

	// Add a root redirect for convenience
	e.GET("/", func(c echo.Context) error {
		return c.Redirect(http.StatusFound, "/dashboard")
	})

	// Start server
	log.Fatalln(e.Start(":8080"))
}
