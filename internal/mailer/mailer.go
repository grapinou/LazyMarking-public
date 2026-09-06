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

	subject := "Subject: Reinitialisation du mot de passe\n"
	body := fmt.Sprintf(
		"Bonjour %s,\n\nUtilisez ce lien pour réinitialiser votre mot de passe :\n%s\n\nSi vous n’avez pas demandé cette réinitialisation, ignorez cet e-mail.\n",
		username, resetLink,
	)
	msg := []byte(subject + "\n" + body)

	return smtp.SendMail(smtpHost+":"+smtpPort, auth, from, []string{toEmail}, msg)
}
