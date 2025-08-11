package handler

import (
	"log"
	"net/http"

	"alc/model"
	"alc/repository"
	"alc/view"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
)

type DashboardHandler struct {
	Repo *repository.Queries
}

func (h *DashboardHandler) ShowDashboard(c echo.Context) error {
	user, ok := c.Get("user").(model.AuthenticatedUser)
	if !ok {
		return c.Redirect(http.StatusFound, "/login")
	}

	ctx := c.Request().Context()
	props := view.DashboardPageProps{
		User: user,
	}

	if user.Role == repository.UserRoleADMIN {
		stats, err := h.Repo.GetDashboardStats(ctx)
		if err != nil {
			log.Printf("Error getting admin stats: %v", err)
			// Non-critical error, can still render the page without stats
		}
		props.AdminStats = stats
	} else { // TECNICO
		pgxUserID := pgtype.UUID{Bytes: user.ID, Valid: true}
		certs, err := h.Repo.GetRecentCertificatesByTechnician(ctx, pgxUserID)
		if err != nil {
			log.Printf("Error getting recent certs for user %s: %v", user.ID, err)
			// Non-critical error, can still render the page
		}
		props.RecentCerts = certs
	}

	return render(c, http.StatusOK, view.DashboardPage(props))
}
