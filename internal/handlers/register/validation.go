package register

import (
	"errors"
	"net/mail"
	"regexp"

	"github.com/grapinou/LazyMarking/internal/handlers/tools"
)

var usernamePattern = regexp.MustCompile(`^[[:alnum:]_.-]{3,64}$`)

func validateRegistration(username, email, password string) error {
	if !usernamePattern.MatchString(username) {
		return errors.New("Le nom d’utilisateur doit contenir de 3 à 64 lettres, chiffres, points, tirets ou traits de soulignement.")
	}
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email {
		return errors.New("L’adresse e-mail n’est pas valide.")
	}
	return tools.ValidatePassword(password)
}
