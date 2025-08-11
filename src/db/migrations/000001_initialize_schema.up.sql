
BEGIN;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

/* --- App users --- */

CREATE TYPE user_role AS ENUM ('ADMIN', 'TECNICO');

CREATE TABLE IF NOT EXISTS app_users (
    user_id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    name varchar(255) NOT NULL,
    email varchar(255) UNIQUE NOT NULL,
    hashed_password text NOT NULL,
    role user_role NOT NULL DEFAULT 'TECNICO',
    dni varchar(25) NOT NULL,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS app_sessions (
    session_id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id uuid NOT NULL REFERENCES app_users ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    expires_at timestamptz NOT NULL DEFAULT NOW() + INTERVAL '1 month'
);

/* --- Machine users --- */

CREATE TABLE IF NOT EXISTS machine_users (
    dni varchar(25) PRIMARY KEY,
    personal_code text UNIQUE NOT NULL,
    name text NOT NULL,
    email varchar(255) UNIQUE NOT NULL,
    society text NOT NULL,
    site text NOT NULL,
    area text NOT NULL,
    floor_name text NOT NULL
);

/* --- Machines and devices --- */

CREATE TYPE machine_type AS ENUM ('PC', 'LAPTOP');
CREATE TYPE machine_profile AS ENUM ('ESPECIAL1', 'ESPECIAL2', 'PROCESAMIENTO', 'REGULAR');

CREATE TABLE IF NOT EXISTS machines (
    serial_num text PRIMARY KEY,
    type machine_type NOT NULL,
    mtm text NOT NULL,
    model text NOT NULL,
    plate_num text NOT NULL,
    disk_size text NOT NULL,
    memory_size text NOT NULL,
    processor text NOT NULL,
    profile machine_profile NOT NULL
);

CREATE TYPE device_type AS ENUM ('NEW', 'OLD');
CREATE TYPE device_status AS ENUM ('ASIGNACION', 'RECUPERACION', 'PRESTAMO', 'BACKUP');

CREATE TABLE IF NOT EXISTS devices (
    device_code text PRIMARY KEY,
    machine_serial_num text UNIQUE NOT NULL REFERENCES machines ON DELETE RESTRICT,
    type device_type NOT NULL,
    hostname text NOT NULL,
    status device_status NOT NULL,
    additional_software text NOT NULL
);

/* --- Software --- */

CREATE TABLE IF NOT EXISTS software (
    software_id int PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    name text NOT NULL
);

CREATE TABLE IF NOT EXISTS device_software (
    device_code text NOT NULL REFERENCES devices ON DELETE CASCADE,
    software_id int NOT NULL REFERENCES software ON DELETE CASCADE,
    PRIMARY KEY (device_code, software_id)
);

/* --- Configuration items --- */

CREATE TABLE IF NOT EXISTS configuration_items (
    item_id int PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    name text NOT NULL
);

CREATE TABLE IF NOT EXISTS device_configuration (
    device_code text NOT NULL REFERENCES devices ON DELETE CASCADE,
    item_id int NOT NULL REFERENCES configuration_items ON DELETE CASCADE,
    PRIMARY KEY (device_code, item_id)
);


/* --- Peripherals --- */

CREATE TABLE IF NOT EXISTS peripherals (
    peripheral_id int PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    name text NOT NULL
);

CREATE TABLE IF NOT EXISTS device_peripherals (
    device_code text NOT NULL REFERENCES devices ON DELETE CASCADE,
    peripheral_id int NOT NULL REFERENCES peripherals ON DELETE CASCADE,
    plate_num text NOT NULL,
    serial_num text NOT NULL,
    PRIMARY KEY (device_code, peripheral_id)
);

/* --- Certificate --- */

CREATE TYPE certificate_status AS ENUM ('PENDING', 'CONFIRMED');

CREATE TABLE IF NOT EXISTS alicorp_2025_certificates (
    certificate_id int PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    ticket_name text UNIQUE NOT NULL,

    app_user_id uuid NOT NULL REFERENCES app_users ON DELETE RESTRICT,
    machine_user_dni varchar(25) NOT NULL REFERENCES machine_users ON DELETE RESTRICT,

    new_device_code text NOT NULL REFERENCES devices ON DELETE RESTRICT,
    old_device_code text NOT NULL REFERENCES devices ON DELETE RESTRICT,

    disk_c_size text NOT NULL,
    disk_d_size text NOT NULL,

    printer_name text NOT NULL,
    printer_ip text NOT NULL,
    printer_test boolean NOT NULL,

    comments text NOT NULL,

    confirmation_status certificate_status NOT NULL DEFAULT 'PENDING',
    confirmation_token uuid UNIQUE DEFAULT uuid_generate_v4(),
    confirmed_at timestamptz,

    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW()
);

COMMIT;

