-- name: GetCertificateForEdit :one
SELECT
    c.*,
    au.name AS technician_name,
    mu.*,
    nd.hostname AS new_device_hostname,
    nd.status AS new_device_status,
    nd.additional_software,
    nm.serial_num AS new_machine_serial,
    nm.type AS new_machine_type,
    nm.model AS new_machine_model,
    nm.disk_size AS new_machine_disk,
    nm.memory_size AS new_machine_memory,
    nm.profile AS new_machine_profile,
    od.hostname AS old_device_hostname,
    od.status AS old_device_status,
    om.serial_num AS old_machine_serial,
    om.type AS old_machine_type,
    om.model AS old_machine_model,
    om.disk_size AS old_machine_disk,
    om.memory_size AS old_machine_memory,
    ARRAY_TO_STRING(ARRAY_AGG(DISTINCT ds.software_id), ',') AS selected_software,
    ARRAY_TO_STRING(ARRAY_AGG(DISTINCT dc.item_id), ',') AS selected_config,
    ARRAY_TO_STRING(ARRAY_AGG(DISTINCT p.peripheral_id || ':' || dp.plate_num || ':' || dp.serial_num), ';') AS selected_peripherals
FROM
    alicorp_2025_certificates c
JOIN app_users au ON c.app_user_id = au.user_id
JOIN machine_users mu ON c.machine_user_dni = mu.dni
JOIN devices nd ON c.new_device_code = nd.device_code
JOIN machines nm ON nd.machine_serial_num = nm.serial_num
LEFT JOIN devices od ON c.old_device_code = od.device_code
LEFT JOIN machines om ON od.machine_serial_num = om.serial_num
LEFT JOIN device_software ds ON nd.device_code = ds.device_code
LEFT JOIN device_configuration dc ON nd.device_code = dc.device_code
LEFT JOIN device_peripherals dp ON nd.device_code = dp.device_code
LEFT JOIN peripherals p ON dp.peripheral_id = p.peripheral_id
WHERE c.certificate_id = $1 AND c.app_user_id = $2
GROUP BY
    c.certificate_id, au.user_id, mu.dni, nd.device_code, nm.serial_num, od.device_code, om.serial_num;

-- name: UpdateCertificate :one
UPDATE alicorp_2025_certificates
SET
    ticket_name = $2,
    machine_user_dni = $3,
    new_device_code = $4,
    old_device_code = $5,
    disk_c_size = $6,
    disk_d_size = $7,
    printer_name = $8,
    printer_ip = $9,
    printer_test = $10,
    comments = $11,
    confirmation_status = 'PENDING', -- Reset status to PENDING
    confirmation_token = uuid_generate_v4(), -- Generate a new token
    updated_at = NOW()
WHERE
    certificate_id = $1 AND app_user_id = $12
RETURNING *;

