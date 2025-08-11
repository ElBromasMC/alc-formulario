package handler

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"alc/repository"
	"alc/view"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

type AdminHandler struct {
	Repo   *repository.Queries
	DBPool *pgxpool.Pool
}

// ShowAdminDashboard now fetches all lists needed for the admin panel.
func (h *AdminHandler) ShowAdminDashboard(c echo.Context) error {
	ctx := context.Background()

	// Fetch all data in parallel for performance
	errs := make(chan error, 4)
	var users []repository.AppUser
	var software []repository.Software
	var peripherals []repository.Peripheral
	var configItems []repository.ConfigurationItem

	go func() {
		var err error
		users, err = h.Repo.ListAppUsers(ctx)
		errs <- err
	}()
	go func() {
		var err error
		software, err = h.Repo.ListSoftware(ctx)
		errs <- err
	}()
	go func() {
		var err error
		peripherals, err = h.Repo.ListPeripherals(ctx)
		errs <- err
	}()
	go func() {
		var err error
		configItems, err = h.Repo.ListConfigurationItems(ctx)
		errs <- err
	}()

	for i := 0; i < 4; i++ {
		if err := <-errs; err != nil {
			log.Printf("Error fetching data for admin dashboard: %v", err)
			return c.String(http.StatusInternalServerError, "Failed to load admin data.")
		}
	}

	props := view.AdminPageProps{
		Users:       users,
		Software:    software,
		Peripherals: peripherals,
		ConfigItems: configItems,
	}

	return render(c, http.StatusOK, view.AdminPage(props))
}

func (h *AdminHandler) HandleCreateUser(c echo.Context) error {
	ctx := context.Background()

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(c.FormValue("password")), bcrypt.DefaultCost)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to hash password")
	}

	params := repository.CreateAppUserParams{
		Name:           c.FormValue("name"),
		Email:          c.FormValue("email"),
		HashedPassword: string(hashedPassword),
		Role:           repository.UserRole(c.FormValue("role")),
		Dni:            c.FormValue("dni"),
	}

	// Basic validation
	if params.Role != repository.UserRoleADMIN && params.Role != repository.UserRoleTECNICO {
		return c.String(http.StatusBadRequest, "Invalid role specified")
	}

	_, err = h.Repo.CreateAppUser(ctx, params)
	if err != nil {
		// Handle potential duplicate email error, etc.
		return c.String(http.StatusInternalServerError, "Failed to create user")
	}

	return c.Redirect(http.StatusFound, "/admin")
}

// HandleCreateSoftware creates a new software item.
func (h *AdminHandler) HandleCreateSoftware(c echo.Context) error {
	name := c.FormValue("name")
	if name == "" {
		return c.String(http.StatusBadRequest, "Name cannot be empty.")
	}
	_, err := h.Repo.CreateSoftware(context.Background(), name)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to create software.")
	}
	return c.Redirect(http.StatusFound, "/admin")
}

// HandleCreatePeripheral creates a new peripheral item.
func (h *AdminHandler) HandleCreatePeripheral(c echo.Context) error {
	name := c.FormValue("name")
	if name == "" {
		return c.String(http.StatusBadRequest, "Name cannot be empty.")
	}
	_, err := h.Repo.CreatePeripheral(context.Background(), name)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to create peripheral.")
	}
	return c.Redirect(http.StatusFound, "/admin")
}

// HandleCreateConfigurationItem creates a new configuration item.
func (h *AdminHandler) HandleCreateConfigurationItem(c echo.Context) error {
	name := c.FormValue("name")
	if name == "" {
		return c.String(http.StatusBadRequest, "Name cannot be empty.")
	}
	_, err := h.Repo.CreateConfigurationItem(context.Background(), name)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to create configuration item.")
	}
	return c.Redirect(http.StatusFound, "/admin")
}

// HandleBulkUploadMachineUsers performs a transactional upsert for machine users.
func (h *AdminHandler) HandleBulkUploadMachineUsers(c echo.Context) error {
	file, err := c.FormFile("csvfile")
	if err != nil {
		return c.String(http.StatusBadRequest, "Failed to get the file.")
	}
	src, err := file.Open()
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to open the file.")
	}
	defer src.Close()

	reader := csv.NewReader(src)
	records, err := reader.ReadAll()
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to parse CSV.")
	}

	if len(records) < 2 {
		return c.String(http.StatusBadRequest, "CSV file is empty or has only a header.")
	}

	// Begin a transaction
	tx, err := h.DBPool.Begin(c.Request().Context())
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to start database transaction.")
	}
	defer tx.Rollback(c.Request().Context()) // Rollback in case of error

	qtx := h.Repo.WithTx(tx)
	processedCount := 0

	// Skip header row
	for _, row := range records[1:] {
		if len(row) < 9 {
			continue // Skip malformed rows
		}

		email := strings.ToLower(strings.TrimSpace(row[5]))
		if _, err := mail.ParseAddress(email); err != nil {
			log.Printf("Skipping row with invalid email: %s", email)
			continue // Skip invalid email
		}

		params := repository.UpsertMachineUserParams{
			Dni:          strings.TrimSpace(row[2]),
			PersonalCode: strings.TrimSpace(row[1]),
			Name:         strings.TrimSpace(row[3]),
			Email:        email,
			Society:      strings.TrimSpace(row[0]),
			Site:         strings.TrimSpace(row[4]),
			Area:         strings.TrimSpace(row[7]),
			FloorName:    strings.TrimSpace(row[8]),
		}

		if params.Dni == "" || params.Name == "" {
			continue // Skip rows with missing required data
		}

		_, err := qtx.UpsertMachineUser(c.Request().Context(), params)
		if err != nil {
			log.Printf("Failed to upsert user with DNI %s: %v", params.Dni, err)
			return c.String(http.StatusInternalServerError, fmt.Sprintf("Error processing user with DNI %s.", params.Dni))
		}
		processedCount++
	}

	// Commit the transaction if all upserts were successful
	if err := tx.Commit(c.Request().Context()); err != nil {
		return c.String(http.StatusInternalServerError, "Failed to commit transaction.")
	}

	log.Printf("Successfully processed %d machine users.", processedCount)
	return c.Redirect(http.StatusFound, "/admin")
}

// HandleBulkUploadMachines performs a transactional upsert for machines.
func (h *AdminHandler) HandleBulkUploadMachines(c echo.Context) error {
	file, err := c.FormFile("csvfile")
	if err != nil {
		return c.String(http.StatusBadRequest, "Failed to get the file.")
	}
	src, err := file.Open()
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to open the file.")
	}
	defer src.Close()

	reader := csv.NewReader(src)
	records, err := reader.ReadAll()
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to parse CSV.")
	}

	if len(records) < 2 {
		return c.String(http.StatusBadRequest, "CSV file is empty or has only a header.")
	}

	tx, err := h.DBPool.Begin(c.Request().Context())
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to start database transaction.")
	}
	defer tx.Rollback(c.Request().Context())

	qtx := h.Repo.WithTx(tx)
	processedCount := 0

	for _, row := range records[1:] {
		if len(row) < 9 {
			continue
		}

		// Normalize ENUM values
		machineType := repository.MachineType(strings.ToUpper(strings.TrimSpace(row[1])))
		if machineType != repository.MachineTypePC && machineType != repository.MachineTypeLAPTOP {
			log.Printf("Skipping row with invalid machine type: %s", row[1])
			continue
		}

		profileStr := strings.ToUpper(strings.TrimSpace(row[8]))
		var machineProfile repository.MachineProfile
		switch profileStr {
		case "ESPECIAL 1":
			machineProfile = repository.MachineProfileESPECIAL1
		case "ESPECIAL 2":
			machineProfile = repository.MachineProfileESPECIAL2
		case "PROCESAMIENTO":
			machineProfile = repository.MachineProfilePROCESAMIENTO
		case "REGULAR":
			machineProfile = repository.MachineProfileREGULAR
		default:
			log.Printf("Skipping row with invalid profile: %s", profileStr)
			continue
		}

		params := repository.UpsertMachineParams{
			SerialNum:  strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(row[0]), " ", "")),
			Type:       machineType,
			Mtm:        strings.TrimSpace(row[2]),
			Model:      strings.TrimSpace(row[3]),
			PlateNum:   strings.TrimSpace(row[4]),
			DiskSize:   strings.TrimSpace(row[5]),
			MemorySize: strings.TrimSpace(row[6]),
			Processor:  strings.TrimSpace(row[7]),
			Profile:    machineProfile,
		}

		if params.SerialNum == "" {
			continue
		}

		_, err := qtx.UpsertMachine(c.Request().Context(), params)
		if err != nil {
			log.Printf("Failed to upsert machine with S/N %s: %v", params.SerialNum, err)
			return c.String(http.StatusInternalServerError, fmt.Sprintf("Error processing machine with S/N %s.", params.SerialNum))
		}
		processedCount++
	}

	if err := tx.Commit(c.Request().Context()); err != nil {
		return c.String(http.StatusInternalServerError, "Failed to commit transaction.")
	}

	log.Printf("Successfully processed %d machines.", processedCount)
	return c.Redirect(http.StatusFound, "/admin")
}

// HandleDownloadReport generates and serves the certificate report as a CSV file.
func (h *AdminHandler) HandleDownloadReport(c echo.Context) error {
	ctx := c.Request().Context()
	reportData, err := h.Repo.GetCertificatesReport(ctx)
	if err != nil {
		log.Printf("Error fetching certificate report: %v", err)
		return c.String(http.StatusInternalServerError, "Could not generate report.")
	}

	// Set headers to trigger browser download
	fileName := fmt.Sprintf("reporte_certificados_%s.csv", time.Now().Format("20060102"))
	c.Response().Header().Set("Content-Type", "text/csv")
	c.Response().Header().Set("Content-Disposition", "attachment; filename="+fileName)

	writer := csv.NewWriter(c.Response().Writer)

	// Write CSV Header
	header := []string{
		"ID Certificado", "Ticket", "Estado", "Fecha Creación", "Técnico", "Email Técnico",
		"DNI Usuario", "Cod. Personal Usuario", "Nombre Usuario", "Email Usuario", "Sociedad", "Sede", "Área", "Piso",
		"Cod. Equipo Nuevo", "Hostname Nuevo", "Estado Nuevo", "Serial Nuevo", "Tipo Nuevo", "Modelo Nuevo", "Disco Nuevo", "RAM Nueva", "Perfil Nuevo",
		"Cod. Equipo Antiguo", "Hostname Antiguo", "Serial Antiguo", "Tipo Antiguo", "Modelo Antiguo",
		"Software", "Configuración", "Periféricos",
		"Tamaño Disco C", "Tamaño Disco D", "Impresora", "IP Impresora", "Test Impresión OK", "Comentarios",
	}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Write data rows
	for _, row := range reportData {
		record := []string{
			fmt.Sprintf("%d", row.CertificateID),
			row.TicketName,
			string(row.ConfirmationStatus),
			row.CertificateCreatedAt.Time.Format(time.RFC3339),
			row.TechnicianName,
			row.TechnicianEmail,
			row.UserDni,
			row.UserPersonalCode,
			row.UserName,
			row.UserEmail,
			row.UserSociety,
			row.UserSite,
			row.UserArea,
			row.UserFloor,
			row.NewDeviceCode,
			row.NewDeviceHostname,
			string(row.NewDeviceStatus),
			row.NewMachineSerial,
			string(row.NewMachineType),
			row.NewMachineModel,
			row.NewMachineDisk,
			row.NewMachineMemory,
			string(row.NewMachineProfile),
			row.OldDeviceCode.String,
			row.OldDeviceHostname.String,
			row.OldMachineSerial.String,
			string(row.OldMachineType.MachineType),
			row.OldMachineModel.String,
			row.SoftwareList,
			row.ConfigItemList,
			row.PeripheralList,
			row.DiskCSize,
			row.DiskDSize,
			row.PrinterName,
			row.PrinterIp,
			fmt.Sprintf("%t", row.PrinterTest),
			row.Comments,
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	writer.Flush()
	return nil
}
