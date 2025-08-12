package handler

import (
	"alc/model"
	"alc/repository"
	"alc/service"
	"alc/view"
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
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

	// Timezone
	LimaLocation, _ := time.LoadLocation("America/Lima")

	// Prepare props for the view
	props := view.CertificatePageProps{
		TechnicianName:   user.Name,
		CurrentDate:      time.Now().In(LimaLocation).Format("02/01/2006"),
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

	// Timezone
	LimaLocation, err := time.LoadLocation("America/Lima")

	freshProps := view.CertificatePageProps{
		TechnicianName:   user.Name,
		CurrentDate:      time.Now().In(LimaLocation).Format("02/01/2006"),
		StandardSoftware: standardSoftware,
		StandardConfig:   standardConfig,
		Peripherals:      peripherals,
	}

	return render(c, http.StatusOK, view.CertificateSubmissionSuccess(freshProps))
}

func (h *CertificateHandler) HandleCertificateConfirmation(c echo.Context) error {
	tokenStr := c.Param("token")
	token, err := uuid.Parse(tokenStr)
	if err != nil {
		return render(c, http.StatusBadRequest, view.ConfirmationResultPage("Error", "El enlace utilizado es inválido."))
	}

	ctx := c.Request().Context()
	pgxToken := pgtype.UUID{Bytes: token, Valid: true}

	// First, get the certificate to check its status
	cert, err := h.Repo.GetCertificateByToken(ctx, pgxToken)
	if err != nil {
		return render(c, http.StatusNotFound, view.ConfirmationResultPage("Error", "El certificado no fue encontrado."))
	}

	// Check if already processed
	if cert.ConfirmationStatus != repository.CertificateStatusPENDING {
		return render(c, http.StatusOK, view.ConfirmationResultPage("Aviso", fmt.Sprintf("Esta solicitud ya fue marcada como %s.", cert.ConfirmationStatus)))
	}

	// If pending, update the status
	err = h.Repo.UpdateCertificateStatus(ctx, repository.UpdateCertificateStatusParams{
		ConfirmationStatus: repository.CertificateStatusCONFIRMED,
		ConfirmationToken:  pgxToken,
	})
	if err != nil {
		log.Printf("Error confirming certificate with token %s: %v", tokenStr, err)
		return render(c, http.StatusInternalServerError, view.ConfirmationResultPage("Error", "No se pudo procesar la confirmación."))
	}

	// On success, send the final email
	go func() {
		user, err := h.Repo.GetMachineUserByDNI(context.Background(), cert.MachineUserDni)
		if err != nil {
			log.Printf("Could not get machine user to send final email: %v", err)
			return
		}
		if err := h.CertSvc.EmailSvc.SendFinalCertificateEmail(context.Background(), user, cert); err != nil {
			log.Printf("Failed to send final certificate email: %v", err)
		}
	}()

	return render(c, http.StatusOK, view.ConfirmationResultPage("¡Gracias!", "Tu conformidad ha sido registrada con éxito."))
}

func (h *CertificateHandler) HandleCertificateRejection(c echo.Context) error {
	tokenStr := c.Param("token")
	token, err := uuid.Parse(tokenStr)
	if err != nil {
		return render(c, http.StatusBadRequest, view.ConfirmationResultPage("Error", "El enlace utilizado es inválido."))
	}

	ctx := c.Request().Context()
	pgxToken := pgtype.UUID{Bytes: token, Valid: true}

	// First, get the certificate to check its status
	cert, err := h.Repo.GetCertificateByToken(ctx, pgxToken)
	if err != nil {
		return render(c, http.StatusNotFound, view.ConfirmationResultPage("Error", "El certificado no fue encontrado."))
	}

	// Check if already processed
	if cert.ConfirmationStatus != repository.CertificateStatusPENDING {
		return render(c, http.StatusOK, view.ConfirmationResultPage("Aviso", fmt.Sprintf("Esta solicitud ya fue marcada como %s.", cert.ConfirmationStatus)))
	}

	err = h.Repo.UpdateCertificateStatus(ctx, repository.UpdateCertificateStatusParams{
		ConfirmationStatus: repository.CertificateStatusREJECTED,
		ConfirmationToken:  pgxToken,
	})
	if err != nil {
		log.Printf("Error rejecting certificate with token %s: %v", tokenStr, err)
		return render(c, http.StatusInternalServerError, view.ConfirmationResultPage("Error", "No se pudo procesar la observación."))
	}

	return render(c, http.StatusOK, view.ConfirmationResultPage("Procesado", "Tu observación ha sido registrada."))
}

func parsePeripherals(data string) map[string]map[string]string {
	result := make(map[string]map[string]string)
	if data == "" {
		return result
	}
	peripherals := strings.Split(data, "; ")
	for _, p := range peripherals {
		parts := strings.SplitN(p, " (", 2)
		if len(parts) < 2 {
			continue
		}
		name := parts[0]
		details := strings.TrimSuffix(parts[1], ")")

		result[name] = make(map[string]string)
		detailParts := strings.Split(details, ", ")
		for _, dp := range detailParts {
			kv := strings.SplitN(dp, ": ", 2)
			if len(kv) == 2 {
				result[name][kv[0]] = kv[1]
			}
		}
	}
	return result
}

func (h *CertificateHandler) ShowCertificate(c echo.Context) error {
	tokenStr := c.Param("token")
	token, err := uuid.Parse(tokenStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "Token inválido.")
	}

	ctx := c.Request().Context()
	pgxToken := pgtype.UUID{Bytes: token, Valid: true}

	certDetails, _ := h.Repo.GetCertificateDetailsByToken(ctx, pgxToken)
	allSoftware, _ := h.Repo.ListSoftware(ctx)
	allConfigItems, _ := h.Repo.ListConfigurationItems(ctx)
	allPeripherals, _ := h.Repo.ListPeripherals(ctx)

	props := view.ViewCertificatePageProps{
		Cert:           certDetails,
		AllSoftware:    allSoftware,
		AllConfig:      allConfigItems,
		AllPeripherals: allPeripherals,
		PeripheralMap:  parsePeripherals(certDetails.PeripheralList),
	}

	return render(c, http.StatusOK, view.ViewCertificatePage(props))
}

func parseEditPeripherals(data string) map[string]map[string]string {
	result := make(map[string]map[string]string)
	if data == "" {
		return result
	}
	peripherals := strings.Split(data, ";")
	for _, p := range peripherals {
		// e.g., "1:123:456" (ID:Plate:SN)
		parts := strings.Split(p, ":")
		if len(parts) == 3 {
			id := parts[0]
			result[id] = map[string]string{
				"Plate": parts[1],
				"SN":    parts[2],
			}
		}
	}
	return result
}

// ShowEditCertificateForm fetches all data for a rejected certificate and displays the edit form.
func (h *CertificateHandler) ShowEditCertificateForm(c echo.Context) error {
	user, ok := c.Get("user").(model.AuthenticatedUser)
	if !ok {
		return c.Redirect(http.StatusFound, "/login")
	}

	certID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "ID de certificado inválido.")
	}

	ctx := c.Request().Context()
	pgxUserID := pgtype.UUID{Bytes: user.ID, Valid: true}

	certData, err := h.Repo.GetCertificateForEdit(ctx, repository.GetCertificateForEditParams{
		CertificateID: int32(certID),
		AppUserID:     pgxUserID,
	})
	if err != nil {
		return c.String(http.StatusBadRequest, "No se encuentra el certificado para editar.")
	}

	allSoftware, _ := h.Repo.ListSoftware(ctx)
	allConfigItems, _ := h.Repo.ListConfigurationItems(ctx)
	allPeripherals, _ := h.Repo.ListPeripherals(ctx)

	// Prepare props for the view
	props := view.CertificateEditPageProps{
		CertData:       certData,
		AllSoftware:    allSoftware,
		AllConfig:      allConfigItems,
		AllPeripherals: allPeripherals,
		PeripheralMap:  parseEditPeripherals(certData.SelectedPeripherals),
	}

	return render(c, http.StatusOK, view.CertificateEditPage(props))
}

// HandleUpdateCertificate processes the submission of the edit form.
func (h *CertificateHandler) HandleUpdateCertificate(c echo.Context) error {
	user, ok := c.Get("user").(model.AuthenticatedUser)
	if !ok {
		c.Response().Header().Set("HX-Retarget", "#form-feedback")
		return render(c, http.StatusOK, view.FormError("Error de autenticación."))
	}

	certID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Response().Header().Set("HX-Retarget", "#form-feedback")
		return render(c, http.StatusOK, view.FormError("ID de certificado inválido."))
	}

	formValues, err := c.FormParams()
	if err != nil {
		c.Response().Header().Set("HX-Retarget", "#form-feedback")
		return render(c, http.StatusOK, view.FormError("Error al procesar el formulario."))
	}

	// Call the update service
	_, err = h.CertSvc.UpdateCertificateFromForm(c.Request().Context(), user, int32(certID), formValues)
	if err != nil {
		log.Printf("ERROR updating certificate: %v", err)
		c.Response().Header().Set("HX-Retarget", "#form-feedback")
		return render(c, http.StatusOK, view.FormError("Error al actualizar: "+err.Error()))
	}

	// On success, tell the frontend to redirect to the dashboard
	c.Response().Header().Set("HX-Redirect", "/dashboard")
	return c.NoContent(http.StatusOK)
}

func (h *CertificateHandler) ShowConfirmationActionPage(c echo.Context) error {
	tokenStr := c.Param("token")
	choice := c.QueryParam("choice")
	token, err := uuid.Parse(tokenStr)
	if err != nil {
		return render(c, http.StatusBadRequest, view.ConfirmationResultPage("Error", "El enlace utilizado es inválido."))
	}

	pgxToken := pgtype.UUID{Bytes: token, Valid: true}
	certDetails, err := h.Repo.GetCertificateDetailsByToken(c.Request().Context(), pgxToken)
	if err != nil {
		return render(c, http.StatusNotFound, view.ConfirmationResultPage("Error", "Certificado no encontrado."))
	}

	props := view.ConfirmationActionPageProps{
		Cert:   certDetails,
		Choice: choice,
	}

	return render(c, http.StatusOK, view.ConfirmationActionPage(props))
}
