package mailer

import (
	"fmt"
	"net/smtp"
	"os"
)

func SendResetEmail(username, toEmail, resetLink string) error {
	from := os.Getenv("SMTP_FROM")
	password := os.Getenv("SMTP_PASSWORD")
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")

	if from == "" || password == "" || smtpHost == "" || smtpPort == "" {
		return fmt.Errorf("missing SMTP configuration")
	}

	auth := smtp.PlainAuth("", from, password, smtpHost)

	subject := "Subject: Password Reset Request\n"
	body := fmt.Sprintf(
		"Hello %s,\n\nClick here to reset your password:\n%s\n\nIf you didn’t request this, ignore this email.\n",
		username, resetLink,
	)
	msg := []byte(subject + "\n" + body)

	return smtp.SendMail(smtpHost+":"+smtpPort, auth, from, []string{toEmail}, msg)
}
