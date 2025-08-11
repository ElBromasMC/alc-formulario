-- name: UpsertMachineUser :one
INSERT INTO machine_users (
    dni, personal_code, name, email, society, site, area, floor_name
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
ON CONFLICT (dni) DO UPDATE SET
    name = EXCLUDED.name,
    email = EXCLUDED.email,
    society = EXCLUDED.society,
    site = EXCLUDED.site,
    area = EXCLUDED.area,
    floor_name = EXCLUDED.floor_name
RETURNING *;

-- name: UpsertMachine :one
INSERT INTO machines (
    serial_num, type, mtm, model, plate_num, disk_size, memory_size, processor, profile
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
)
ON CONFLICT (serial_num) DO UPDATE SET
    type = EXCLUDED.type,
    mtm = EXCLUDED.mtm,
    model = EXCLUDED.model,
    plate_num = EXCLUDED.plate_num,
    disk_size = EXCLUDED.disk_size,
    memory_size = EXCLUDED.memory_size,
    processor = EXCLUDED.processor,
    profile = EXCLUDED.profile
RETURNING *;

