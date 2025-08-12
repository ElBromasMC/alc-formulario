-- name: CreateCertificate :one
INSERT INTO alicorp_2025_certificates (
    ticket_name,
    app_user_id,
    machine_user_dni,
    new_device_code,
    old_device_code,
    disk_c_size,
    disk_d_size,
    printer_name,
    printer_ip,
    printer_test,
    comments
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
) RETURNING *;

-- name: AddSoftwareToDevice :exec
INSERT INTO device_software (device_code, software_id)
VALUES ($1, $2);

-- name: AddConfigToDevice :exec
INSERT INTO device_configuration (device_code, item_id)
VALUES ($1, $2);

-- name: AddPeripheralToDevice :exec
INSERT INTO device_peripherals (device_code, peripheral_id, plate_num, serial_num)
VALUES ($1, $2, $3, $4);

-- name: ListMachineUsers :many
SELECT * FROM machine_users ORDER BY name;

-- name: ListDevices :many
SELECT * FROM devices ORDER BY device_code;

-- name: ListSoftware :many
SELECT * FROM software ORDER BY name;

-- name: ListConfigurationItems :many
SELECT * FROM configuration_items ORDER BY name;

-- name: ListPeripherals :many
SELECT * FROM peripherals ORDER BY name;

-- name: UpsertDevice :one
INSERT INTO devices (
    device_code, machine_serial_num, type, hostname, status, additional_software
) VALUES (
    $1, $2, $3, $4, $5, $6
) ON CONFLICT (device_code) DO UPDATE SET
    machine_serial_num = EXCLUDED.machine_serial_num,
    type = EXCLUDED.type,
    hostname = EXCLUDED.hostname,
    status = EXCLUDED.status,
    additional_software = EXCLUDED.additional_software
RETURNING *;

-- name: ClearDeviceSoftware :exec
DELETE FROM device_software WHERE device_code = $1;

-- name: ClearDeviceConfiguration :exec
DELETE FROM device_configuration WHERE device_code = $1;

-- name: ClearDevicePeripherals :exec
DELETE FROM device_peripherals WHERE device_code = $1;

-- name: GetCertificateByToken :one
SELECT
    c.*,
    m.plate_num AS new_device_plate,
    m.serial_num AS new_device_serial,
    m.model AS new_device_model
FROM
    alicorp_2025_certificates c
JOIN devices d ON c.new_device_code = d.device_code
JOIN machines m ON d.machine_serial_num = m.serial_num
WHERE
    c.confirmation_token = $1;

-- name: UpdateCertificateStatus :exec
UPDATE alicorp_2025_certificates
SET
    confirmation_status = $1,
    confirmed_at = NOW(),
    updated_at = NOW()
WHERE
    confirmation_token = $2 AND confirmation_status = 'PENDING';

