-- Add 'REJECTED' to the certificate_status enum
ALTER TYPE certificate_status ADD VALUE 'REJECTED';

-- Add a UNIQUE constraint to the new_device_code column
-- This ensures a new device can only be assigned in one certificate.
ALTER TABLE alicorp_2025_certificates
ADD CONSTRAINT alicorp_2025_certificates_new_device_code_key UNIQUE (new_device_code);

