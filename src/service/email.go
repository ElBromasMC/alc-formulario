package service

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"log"

	"alc/config"
	"alc/repository"

	"github.com/wneessen/go-mail"
)

const confirmationTpl = `
<!DOCTYPE html>
<html>
<head>
    <title>Confirmación de Asignación de Equipo</title>
</head>
<body style="font-family: Arial, sans-serif;">
    <h2>Confirmación de Asignación de Equipo</h2>
    <p>Hola {{.UserName}},</p>
    <p>Se ha registrado una nueva asignación de equipo a tu nombre. Por favor, revisa los detalles y confirma o rechaza la conformidad.</p>
    <ul>
        <li><strong>Modelo:</strong> {{.NewDeviceModel}}</li>
        <li><strong>N/S:</strong> {{.NewDeviceSerial}}</li>
        <li><strong>Placa:</strong> {{.NewDevicePlate}}</li>
    </ul>
    <p><a href="{{.ViewURL}}" style="padding: 10px 15px; background-color: #007bff; color: white; text-decoration: none; border-radius: 5px;">Ver Acta de Asignación</a></p>
    <p>Para aceptar, por favor haz clic en el siguiente enlace:</p>
    <p><a href="{{.ConfirmURL}}" style="padding: 10px 15px; background-color: #28a745; color: white; text-decoration: none; border-radius: 5px;">Confirmar Asignación</a></p>
    <p>Si no reconoces esta actividad o deseas rechazarla, haz clic aquí:</p>
    <p><a href="{{.RejectURL}}" style="padding: 10px 15px; background-color: #dc3545; color: white; text-decoration: none; border-radius: 5px;">Observar Asignación</a></p>
    <p>Gracias,<br>El equipo de Renovación Tecnológica</p>
</body>
</html>
`

const finalCertificateTpl = `
<!DOCTYPE html>
<html>
<head>
    <title>Acta de Asignación Confirmada</title>
</head>
<body style="font-family: Arial, sans-serif;">
    <h2>Acta de Asignación Confirmada</h2>
    <p>Hola {{.UserName}},</p>
	<p>Tu conformidad para la asignación del siguiente equipo ha sido registrada con éxito:</p>
    <ul>
        <li><strong>Modelo:</strong> {{.NewDeviceModel}}</li>
        <li><strong>N/S:</strong> {{.NewDeviceSerial}}</li>
        <li><strong>Placa:</strong> {{.NewDevicePlate}}</li>
        <li><strong>Firma digital:</strong> {{.DigitalSignature}}</li>
    </ul>
    <p>Puedes ver una copia del acta en cualquier momento haciendo clic en el siguiente enlace:</p>
    <p><a href="{{.ViewURL}}" style="padding: 10px 15px; background-color: #007bff; color: white; text-decoration: none; border-radius: 5px;">Ver Acta de Conformidad</a></p>
    <p>Gracias,<br>El equipo de Renovación Tecnológica</p>
</body>
</html>
`

type EmailService struct {
	config *config.Config
	client *mail.Client
}

func NewEmailService(cfg *config.Config) (*EmailService, error) {
	c, err := mail.NewClient(cfg.SmtpHost, mail.WithPort(cfg.SmtpPort), mail.WithUsername(cfg.SmtpUser), mail.WithPassword(cfg.SmtpPass), mail.WithSMTPAuth(mail.SMTPAuthPlain))
	if err != nil {
		return nil, err
	}
	return &EmailService{config: cfg, client: c}, nil
}

func (s *EmailService) SendConfirmationEmail(ctx context.Context, user repository.MachineUser, cert repository.Alicorp2025Certificate, machine repository.Machine) error {
	msg := mail.NewMsg()
	if err := msg.From(s.config.SmtpSender); err != nil {
		return err
	}
	if err := msg.To(user.Email); err != nil {
		return err
	}
	msg.Subject(fmt.Sprintf("Por favor, confirma la asignación del equipo (Código: %s)", machine.PlateNum))

	// Prepare template data
	data := struct {
		UserName        string
		ViewURL         string
		ConfirmURL      string
		RejectURL       string
		NewDevicePlate  string
		NewDeviceSerial string
		NewDeviceModel  string
	}{
		UserName:        user.Name,
		ViewURL:         fmt.Sprintf("%s/certificate/view/%s", s.config.AppBaseURL, cert.ConfirmationToken.String()),
		ConfirmURL:      fmt.Sprintf("%s/certificate/action/%s?choice=confirm", s.config.AppBaseURL, cert.ConfirmationToken.String()),
		RejectURL:       fmt.Sprintf("%s/certificate/action/%s?choice=reject", s.config.AppBaseURL, cert.ConfirmationToken.String()),
		NewDevicePlate:  machine.PlateNum,
		NewDeviceSerial: machine.SerialNum,
		NewDeviceModel:  machine.Model,
	}

	// Parse and execute template
	t, err := template.New("confirmation").Parse(confirmationTpl)
	if err != nil {
		return err
	}
	var body bytes.Buffer
	if err := t.Execute(&body, data); err != nil {
		return err
	}

	msg.SetBodyString(mail.TypeTextHTML, body.String())

	// Send the email
	if err := s.client.DialAndSend(msg); err != nil {
		return err
	}
	log.Printf("Confirmation email sent successfully to %s", user.Email)
	return nil
}

func (s *EmailService) SendFinalCertificateEmail(ctx context.Context, user repository.MachineUser, cert repository.GetCertificateByTokenRow) error {
	msg := mail.NewMsg()
	if err := msg.From(s.config.SmtpSender); err != nil {
		return err
	}
	if err := msg.To(user.Email); err != nil {
		return err
	}
	msg.Subject(fmt.Sprintf("Acta de conformidad registrada para equipo: %s", cert.NewDevicePlate))

	data := struct {
		UserName         string
		ViewURL          string
		NewDevicePlate   string
		NewDeviceSerial  string
		NewDeviceModel   string
		DigitalSignature string
	}{
		UserName:         user.Name,
		ViewURL:          fmt.Sprintf("%s/certificate/view/%s", s.config.AppBaseURL, cert.ConfirmationToken.String()),
		NewDevicePlate:   cert.NewDevicePlate,
		NewDeviceSerial:  cert.NewDeviceSerial,
		NewDeviceModel:   cert.NewDeviceModel,
		DigitalSignature: cert.ConfirmationToken.String(),
	}

	t, err := template.New("final").Parse(finalCertificateTpl)
	if err != nil {
		return err
	}
	var body bytes.Buffer
	if err := t.Execute(&body, data); err != nil {
		return err
	}
	msg.SetBodyString(mail.TypeTextHTML, body.String())

	if err := s.client.DialAndSend(msg); err != nil {
		return err
	}
	log.Printf("Final certificate email sent successfully to %s", user.Email)
	return nil
}
