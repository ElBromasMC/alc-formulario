package handler

import (
	"context"
	"net/http"
	"strings"

	"alc/repository"
	"alc/view"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
)

type ApiHandler struct {
	Repo *repository.Queries
}

func (h *ApiHandler) GetMachineUser(c echo.Context) error {
	// NORMALIZE: Trim spaces from the DNI
	dni := strings.ReplaceAll(c.QueryParam("machine_user_dni"), " ", "")

	if dni == "" {
		// If the input is cleared, return the NotFound fragment to clear the form
		return render(c, http.StatusOK, view.MachineUserNotFound())
	}

	user, err := h.Repo.GetMachineUserByDNI(context.Background(), dni)
	if err != nil {
		if err == pgx.ErrNoRows {
			// Return the NotFound fragment to clear fields and show a message
			return render(c, http.StatusOK, view.MachineUserNotFound())
		}
		return c.String(http.StatusInternalServerError, "Database error.")
	}

	return render(c, http.StatusOK, view.MachineUserDetails(user))
}

func (h *ApiHandler) GetMachine(c echo.Context) error {
	// NORMALIZE: Trim spaces and convert to uppercase for the serial number
	serial := strings.ToUpper(strings.ReplaceAll(c.QueryParam("new_device_serial"), " ", ""))

	if serial == "" {
		return render(c, http.StatusOK, view.NewDeviceNotFound())
	}

	machine, err := h.Repo.GetMachineBySerial(context.Background(), serial)
	if err != nil {
		if err == pgx.ErrNoRows {
			return render(c, http.StatusOK, view.NewDeviceNotFound())
		}
		return c.String(http.StatusInternalServerError, "Database error.")
	}

	return render(c, http.StatusOK, view.NewDeviceDetails(machine))
}
