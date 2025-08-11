-- name: CreateSoftware :one
INSERT INTO software (name) VALUES ($1)
RETURNING *;

-- name: CreatePeripheral :one
INSERT INTO peripherals (name) VALUES ($1)
RETURNING *;

-- name: CreateConfigurationItem :one
INSERT INTO configuration_items (name) VALUES ($1)
RETURNING *;

