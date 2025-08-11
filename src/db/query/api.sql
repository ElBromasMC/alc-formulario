-- name: GetMachineUserByDNI :one
SELECT * FROM machine_users
WHERE dni = $1;

-- name: GetMachineBySerial :one
SELECT * FROM machines
WHERE serial_num = $1;

