package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	SmtpHost          string
	SmtpPort          int
	SmtpUser          string
	SmtpPass          string
	SmtpSender        string
	SmtpBccRecipients []string
	AppBaseURL        string
}

func Load() (*Config, error) {
	port, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))

	// Read and parse the BCC recipients
	bccStr := os.Getenv("SMTP_BCC_RECIPIENTS")
	var bccList []string
	if bccStr != "" {
		bccList = strings.Split(bccStr, ",")
	}

	return &Config{
		SmtpHost:          os.Getenv("SMTP_HOST"),
		SmtpPort:          port,
		SmtpUser:          os.Getenv("SMTP_USER"),
		SmtpPass:          os.Getenv("SMTP_PASS"),
		SmtpSender:        os.Getenv("SMTP_SENDER"),
		SmtpBccRecipients: bccList,
		AppBaseURL:        os.Getenv("APP_BASE_URL"),
	}, nil
}
