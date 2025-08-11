package handler

import (
	"context"
	"net/http"
	"time"

	"alc/repository"
	"alc/view"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

const AppSessionCookie = "app_session_id"

type AuthHandler struct {
	Repo *repository.Queries
}

func (h *AuthHandler) ShowLoginPage(c echo.Context) error {
	return render(c, http.StatusOK, view.LoginPage("/login", ""))
}

func (h *AuthHandler) HandleLogin(c echo.Context) error {
	ctx := context.Background()
	email := c.FormValue("email")
	password := c.FormValue("password")

	// 1. Find user by email
	user, err := h.Repo.GetAppUserByEmail(ctx, email)
	if err != nil {
		if err == pgx.ErrNoRows {
			return render(c, http.StatusUnauthorized, view.LoginPage("/login", "Invalid email or password."))
		}
		return c.String(http.StatusInternalServerError, "Database error")
	}

	// 2. Compare password with hashed password
	err = bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(password))
	if err != nil {
		// Password does not match
		return render(c, http.StatusUnauthorized, view.LoginPage("/login", "Invalid email or password."))
	}

	// 3. Create a session
	session, err := h.Repo.CreateAppSession(ctx, user.UserID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Could not create session")
	}

	// 4. Set a cookie
	cookie := new(http.Cookie)
	cookie.Name = AppSessionCookie
	cookie.Value = session.SessionID.String()
	cookie.Expires = time.Now().Add(30 * 24 * time.Hour) // 1 month
	cookie.Path = "/"
	cookie.HttpOnly = true
	c.SetCookie(cookie)

	// 5. Redirect to a protected page
	return c.Redirect(http.StatusFound, "/dashboard")
}

func (h *AuthHandler) HandleLogout(c echo.Context) error {
	cookie := new(http.Cookie)
	cookie.Name = AppSessionCookie
	cookie.Value = ""
	cookie.Expires = time.Unix(0, 0)
	cookie.Path = "/"
	cookie.HttpOnly = true
	c.SetCookie(cookie)

	return c.Redirect(http.StatusFound, "/login")
}
