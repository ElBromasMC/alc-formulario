package handler

import (
	"log"
	"net/http"
	"time"

	"alc/model"
	"alc/repository"
	"alc/service"
	"alc/view"

	"github.com/labstack/echo/v4"
)

type CertificateHandler struct {
	Repo    *repository.Queries
	CertSvc *service.CertificateService
}

// ShowCertificateForm fetches all necessary data and renders the certificate creation page.
func (h *CertificateHandler) ShowCertificateForm(c echo.Context) error {
	// Get the authenticated user from the context
	user, ok := c.Get("user").(model.AuthenticatedUser)
	if !ok {
		return c.Redirect(http.StatusFound, "/login")
	}

	ctx := c.Request().Context()
	standardSoftware, _ := h.Repo.ListSoftware(ctx)
	standardConfig, _ := h.Repo.ListConfigurationItems(ctx)
	peripherals, _ := h.Repo.ListPeripherals(ctx)

	// Prepare props for the view
	props := view.CertificatePageProps{
		TechnicianName:   user.Name,
		CurrentDate:      time.Now().Format("02/01/2006"),
		StandardSoftware: standardSoftware,
		StandardConfig:   standardConfig,
		Peripherals:      peripherals,
	}

	return render(c, http.StatusOK, view.CertificateForm(props))
}

func (h *CertificateHandler) HandleCreateCertificate(c echo.Context) error {
	// Get the authenticated user from the context
	user, ok := c.Get("user").(model.AuthenticatedUser)
	if !ok {
		c.Response().Header().Set("HX-Retarget", "#form-feedback")
		c.Response().Header().Set("HX-Reswap", "innerHTML")
		return render(c, http.StatusOK, view.FormError("Error de autenticación, por favor inicie sesión de nuevo."))
	}

	formValues, err := c.FormParams()
	if err != nil {
		c.Response().Header().Set("HX-Retarget", "#form-feedback")
		c.Response().Header().Set("HX-Reswap", "innerHTML")
		return render(c, http.StatusOK, view.FormError("Error al procesar el formulario."))
	}

	// Call the service to handle all business logic
	cert, err := h.CertSvc.CreateCertificateFromForm(c.Request().Context(), user, formValues)
	if err != nil {
		log.Printf("ERROR creating certificate: %v", err)
		c.Response().Header().Set("HX-Retarget", "#form-feedback")
		c.Response().Header().Set("HX-Reswap", "innerHTML")
		return render(c, http.StatusOK, view.FormError("Error al guardar: "+err.Error()))
	}

	log.Printf("Successfully created certificate with ID: %d", cert.CertificateID)

	// --- SUCCESS LOGIC ---

	ctx := c.Request().Context()
	standardSoftware, _ := h.Repo.ListSoftware(ctx)
	standardConfig, _ := h.Repo.ListConfigurationItems(ctx)
	peripherals, _ := h.Repo.ListPeripherals(ctx)

	freshProps := view.CertificatePageProps{
		TechnicianName:   user.Name,
		CurrentDate:      time.Now().Format("02/01/2006"),
		StandardSoftware: standardSoftware,
		StandardConfig:   standardConfig,
		Peripherals:      peripherals,
	}

	return render(c, http.StatusOK, view.CertificateSubmissionSuccess(freshProps))
}
