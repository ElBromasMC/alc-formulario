package service

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"strconv"
	"strings"

	"alc/model"
	"alc/repository"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CertificateService struct {
	DBPool *pgxpool.Pool
	Repo   *repository.Queries
}

func NewCertificateService(db *pgxpool.Pool, r *repository.Queries) *CertificateService {
	return &CertificateService{
		DBPool: db,
		Repo:   r,
	}
}

// Helper function to normalize strings
func normalize(s string, toUpper bool) string {
	s = strings.TrimSpace(s)
	if toUpper {
		s = strings.ToUpper(s)
	}
	return s
}

// CreateCertificateFromForm orchestrates the entire process in a single transaction.
func (s *CertificateService) CreateCertificateFromForm(ctx context.Context, user model.AuthenticatedUser, form url.Values) (*repository.Alicorp2025Certificate, error) {
	// --- 1. DATA VALIDATION AND NORMALIZATION ---

	// Critical fields
	newDeviceCode := normalize(form.Get("new_device_code"), true)
	if newDeviceCode == "" {
		return nil, errors.New("el 'Código Equipo' del equipo asignado no puede estar vacío")
	}
	newSerial := normalize(form.Get("new_device_serial"), true)
	if newSerial == "" {
		return nil, errors.New("el 'Número de Serie' del equipo asignado no puede estar vacío")
	}
	oldDeviceCode := normalize(form.Get("old_device_code"), true)
	if oldDeviceCode == "" {
		return nil, errors.New("el 'Código Equipo' del equipo liberado no puede estar vacío")
	}
	oldSerial := normalize(form.Get("old_device_serial"), true)
	if oldSerial == "" {
		return nil, errors.New("el 'Número de Serie' del equipo liberado no puede estar vacío")
	}
	userDNI := normalize(form.Get("machine_user_dni"), true)
	if userDNI == "" {
		return nil, errors.New("el 'Código de Usuario' (DNI) no puede estar vacío")
	}

	if newSerial == oldSerial {
		return nil, errors.New("el 'Número de Serie' del equipo asignado y liberado deben ser diferentes")
	}

	if newDeviceCode == oldDeviceCode {
		return nil, errors.New("el 'Código Equipo' del equipo asignado y liberado deben ser diferentes")
	}

	// Email validation and normalization
	userEmail := strings.ToLower(strings.TrimSpace(form.Get("machine_user_email")))
	if userEmail != "" {
		if _, err := mail.ParseAddress(userEmail); err != nil {
			return nil, fmt.Errorf("el formato del correo '%s' no es válido", userEmail)
		}
	}

	// --- 2. DATABASE TRANSACTION ---

	tx, err := s.DBPool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := s.Repo.WithTx(tx)

	// --- 3. Upsert Machine User ---

	machineUser, err := qtx.UpsertMachineUser(ctx, repository.UpsertMachineUserParams{
		Dni:          userDNI,
		PersonalCode: normalize(form.Get("machine_user_code"), true),
		Name:         normalize(form.Get("machine_user_name"), true),
		Email:        userEmail,
		Society:      normalize(form.Get("machine_user_society"), true),
		Site:         normalize(form.Get("machine_user_site"), true),
		Area:         normalize(form.Get("machine_user_area"), true),
		FloorName:    normalize(form.Get("machine_user_floor"), true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upsert machine user: %w", err)
	}

	// --- 4. Upsert NEW Machine ---

	_, err = qtx.UpsertMachine(ctx, repository.UpsertMachineParams{
		SerialNum:  newSerial,
		Type:       repository.MachineType(normalize(form.Get("new_device_type"), true)),
		Model:      normalize(form.Get("new_device_model"), false),
		PlateNum:   newDeviceCode,
		DiskSize:   normalize(form.Get("new_device_disk"), false),
		MemorySize: normalize(form.Get("new_device_memory"), false),
		Profile:    repository.MachineProfile(normalize(form.Get("new_device_profile"), true)),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upsert new machine: %w", err)
	}

	// --- 5. Upsert NEW Device ---

	_, err = qtx.UpsertDevice(ctx, repository.UpsertDeviceParams{
		DeviceCode:         newDeviceCode,
		MachineSerialNum:   newSerial,
		Type:               repository.DeviceTypeNEW,
		Hostname:           normalize(form.Get("new_device_hostname"), true),
		Status:             repository.DeviceStatus(normalize(form.Get("new_device_status"), true)),
		AdditionalSoftware: strings.TrimSpace(form.Get("additional_software")),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upsert new device: %w", err)
	}

	// --- 6. Upsert OLD Machine ---

	_, err = qtx.UpsertMachine(ctx, repository.UpsertMachineParams{
		SerialNum:  oldSerial,
		Type:       repository.MachineType(normalize(form.Get("old_device_type"), true)),
		Model:      normalize(form.Get("old_device_model"), false),
		PlateNum:   oldDeviceCode,
		DiskSize:   normalize(form.Get("old_device_disk"), false),
		MemorySize: normalize(form.Get("old_device_memory"), false),
		Profile:    repository.MachineProfileREGULAR,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upsert old machine: %w", err)
	}

	// --- 7. Upsert OLD Device ---

	_, err = qtx.UpsertDevice(ctx, repository.UpsertDeviceParams{
		DeviceCode:         oldDeviceCode,
		MachineSerialNum:   oldSerial,
		Type:               repository.DeviceTypeOLD,
		Hostname:           normalize(form.Get("old_device_hostname"), false),
		Status:             repository.DeviceStatus(normalize(form.Get("old_device_status"), true)),
		AdditionalSoftware: "",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upsert old device: %w", err)
	}

	// --- 8. Clear and add many-to-many relationships for the new device ---

	// Software
	if err := qtx.ClearDeviceSoftware(ctx, newDeviceCode); err != nil {
		return nil, fmt.Errorf("failed to clear old device software links: %w", err)
	}
	for _, softwareIDStr := range form["standard_software"] {
		softwareID, err := strconv.Atoi(softwareIDStr)
		if err != nil {
			continue // Skip if the value is not a valid integer
		}
		addSoftwareParams := repository.AddSoftwareToDeviceParams{
			DeviceCode: newDeviceCode,
			SoftwareID: int32(softwareID),
		}
		if err := qtx.AddSoftwareToDevice(ctx, addSoftwareParams); err != nil {
			return nil, fmt.Errorf("failed to add software link for ID %d: %w", softwareID, err)
		}
	}

	// Configuration Items
	if err := qtx.ClearDeviceConfiguration(ctx, newDeviceCode); err != nil {
		return nil, fmt.Errorf("failed to clear old device configuration links: %w", err)
	}
	for _, itemIDStr := range form["standard_config"] {
		itemID, err := strconv.Atoi(itemIDStr)
		if err != nil {
			continue // Skip if the value is not a valid integer
		}
		addConfigParams := repository.AddConfigToDeviceParams{
			DeviceCode: newDeviceCode,
			ItemID:     int32(itemID),
		}
		if err := qtx.AddConfigToDevice(ctx, addConfigParams); err != nil {
			return nil, fmt.Errorf("failed to add config link for ID %d: %w", itemID, err)
		}
	}

	// Peripherals
	if err := qtx.ClearDevicePeripherals(ctx, newDeviceCode); err != nil {
		return nil, fmt.Errorf("failed to clear old device peripheral links: %w", err)
	}
	for key, values := range form {
		if strings.HasPrefix(key, "peripheral_plate_") {
			peripheralIDStr := strings.TrimPrefix(key, "peripheral_plate_")
			peripheralID, err := strconv.Atoi(peripheralIDStr)
			if err != nil {
				continue // Skip if the key is malformed
			}

			plateNum := normalize(values[0], true)
			serialNum := normalize(form.Get("peripheral_sn_"+peripheralIDStr), true)

			// Only add the peripheral if it has a plate or serial number
			if plateNum != "" || serialNum != "" {
				addPeripheralParams := repository.AddPeripheralToDeviceParams{
					DeviceCode:   newDeviceCode,
					PeripheralID: int32(peripheralID),
					PlateNum:     plateNum,
					SerialNum:    serialNum,
				}
				if err := qtx.AddPeripheralToDevice(ctx, addPeripheralParams); err != nil {
					return nil, fmt.Errorf("failed to add peripheral link for ID %d: %w", peripheralID, err)
				}
			}
		}
	}

	// --- 9. Create the Certificate ---

	cert, err := qtx.CreateCertificate(ctx, repository.CreateCertificateParams{
		TicketName:     normalize(form.Get("ticket_name"), false),
		AppUserID:      pgtype.UUID{Bytes: user.ID, Valid: true},
		MachineUserDni: machineUser.Dni,
		NewDeviceCode:  newDeviceCode,
		OldDeviceCode:  oldDeviceCode,
		DiskCSize:      normalize(form.Get("disk_c_size"), false),
		DiskDSize:      normalize(form.Get("disk_d_size"), false),
		PrinterName:    normalize(form.Get("printer_name"), false),
		PrinterIp:      normalize(form.Get("printer_ip"), false),
		PrinterTest:    form.Get("printer_test") == "on",
		Comments:       strings.TrimSpace(form.Get("comments")),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	// If all operations were successful, commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &cert, nil
}
