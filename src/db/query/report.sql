-- name: GetCertificatesReport :many
SELECT
    c.certificate_id,
    c.ticket_name,
    c.confirmation_status,
    c.created_at AS certificate_created_at,
    au.name AS technician_name,
    au.email AS technician_email,
    mu.dni AS user_dni,
    mu.personal_code AS user_personal_code,
    mu.name AS user_name,
    mu.email AS user_email,
    mu.society AS user_society,
    mu.site AS user_site,
    mu.area AS user_area,
    mu.floor_name AS user_floor,
    -- New Device Info
    nd.device_code AS new_device_code,
    nd.hostname AS new_device_hostname,
    nd.status AS new_device_status,
    nm.serial_num AS new_machine_serial,
    nm.type AS new_machine_type,
    nm.model AS new_machine_model,
    nm.disk_size AS new_machine_disk,
    nm.memory_size AS new_machine_memory,
    nm.profile AS new_machine_profile,
    -- Old Device Info
    od.device_code AS old_device_code,
    od.hostname AS old_device_hostname,
    om.serial_num AS old_machine_serial,
    om.type AS old_machine_type,
    om.model AS old_machine_model,
    -- Aggregated Data
    ARRAY_TO_STRING(ARRAY_AGG(DISTINCT s.name), ', ') AS software_list,
    ARRAY_TO_STRING(ARRAY_AGG(DISTINCT ci.name), ', ') AS config_item_list,
    ARRAY_TO_STRING(ARRAY_AGG(DISTINCT p.name || ' (Placa: ' || dp.plate_num || ', S/N: ' || dp.serial_num || ')'), '; ') AS peripheral_list,
    -- Other Certificate Data
    c.disk_c_size,
    c.disk_d_size,
    c.printer_name,
    c.printer_ip,
    c.printer_test,
    c.comments
FROM
    alicorp_2025_certificates c
JOIN app_users au ON c.app_user_id = au.user_id
JOIN machine_users mu ON c.machine_user_dni = mu.dni
-- Joins for New Device
JOIN devices nd ON c.new_device_code = nd.device_code
JOIN machines nm ON nd.machine_serial_num = nm.serial_num
-- Joins for Old Device
LEFT JOIN devices od ON c.old_device_code = od.device_code
LEFT JOIN machines om ON od.machine_serial_num = om.serial_num
-- Joins for Many-to-Many relationships (on New Device)
LEFT JOIN device_software ds ON nd.device_code = ds.device_code
LEFT JOIN software s ON ds.software_id = s.software_id
LEFT JOIN device_configuration dc ON nd.device_code = dc.device_code
LEFT JOIN configuration_items ci ON dc.item_id = ci.item_id
LEFT JOIN device_peripherals dp ON nd.device_code = dp.device_code
LEFT JOIN peripherals p ON dp.peripheral_id = p.peripheral_id
GROUP BY
    c.certificate_id, au.user_id, mu.dni, nd.device_code, nm.serial_num, od.device_code, om.serial_num
ORDER BY
    c.created_at DESC;

