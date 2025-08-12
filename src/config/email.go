package config

import (
	"os"
	"strconv"
)

type Config struct {
	SmtpHost   string
	SmtpPort   int
	SmtpUser   string
	SmtpPass   string
	SmtpSender string
	AppBaseURL string
}

func Load() (*Config, error) {
	port, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))

	return &Config{
		SmtpHost:   os.Getenv("SMTP_HOST"),
		SmtpPort:   port,
		SmtpUser:   os.Getenv("SMTP_USER"),
		SmtpPass:   os.Getenv("SMTP_PASS"),
		SmtpSender: os.Getenv("SMTP_SENDER"),
		AppBaseURL: os.Getenv("APP_BASE_URL"),
	}, nil
}
