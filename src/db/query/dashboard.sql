-- name: GetDashboardStats :one
SELECT
    (SELECT COUNT(*) FROM alicorp_2025_certificates) AS total_certificates,
    (SELECT COUNT(*) FROM app_users) AS total_app_users,
    (SELECT COUNT(*) FROM machine_users) AS total_machine_users;

-- name: GetRecentCertificatesByTechnician :many
SELECT
    c.certificate_id,
    c.ticket_name,
    c.created_at,
    c.confirmation_status,
    mu.name as machine_user_name
FROM
    alicorp_2025_certificates c
JOIN
    machine_users mu ON c.machine_user_dni = mu.dni
WHERE
    c.app_user_id = $1
ORDER BY
    c.created_at DESC
LIMIT 5;

