package handler

import (
	"context"
	"net/http"

	"alc/model"
	"alc/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
)

func RequireAuth(repo *repository.Queries) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cookie, err := c.Cookie(AppSessionCookie)
			if err != nil || cookie.Value == "" {
				return c.Redirect(http.StatusFound, "/login")
			}

			// 1. Parse the cookie string into a standard `google/uuid.UUID`
			sessionID, err := uuid.Parse(cookie.Value)
			if err != nil {
				return c.Redirect(http.StatusFound, "/login")
			}

			// 2. CONVERT from `google/uuid.UUID` to `pgtype.UUID` for the database query
			pgxSessionID := pgtype.UUID{Bytes: sessionID, Valid: true}
			user, err := repo.GetAppUserBySessionID(context.Background(), pgxSessionID)
			if err != nil {
				// Invalid or expired session
				return c.Redirect(http.StatusFound, "/login")
			}

			// Store user in context for downstream handlers
			c.Set("user", model.AuthenticatedUser{
				// 3. CONVERT from `pgtype.UUID` back to `google/uuid.UUID` for your model
				ID:    user.UserID.Bytes,
				Name:  user.Name,
				Email: user.Email,
				Role:  user.Role,
			})

			return next(c)
		}
	}
}

// RequireAdmin checks if the user in context has the ADMIN role.
// It must be used AFTER the RequireAuth middleware.
func RequireAdmin() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user, ok := c.Get("user").(model.AuthenticatedUser)
			if !ok {
				// This should not happen if RequireAuth is used first
				return c.Redirect(http.StatusFound, "/login")
			}

			if user.Role != repository.UserRoleADMIN {
				// You can redirect to a dedicated "unauthorized" page
				// or just back to the dashboard.
				return c.Redirect(http.StatusFound, "/dashboard")
			}

			return next(c)
		}
	}
}
