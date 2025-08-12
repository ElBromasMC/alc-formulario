-- Remove the UNIQUE constraint from the new_device_code column
ALTER TABLE alicorp_2025_certificates
DROP CONSTRAINT alicorp_2025_certificates_new_device_code_key;

