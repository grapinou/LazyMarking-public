package register

import (
	"errors"
	"net/mail"
	"regexp"
)

var usernamePattern = regexp.MustCompile(`^[[:alnum:]_.-]{3,64}$`)

func validateRegistration(username, email, password string) error {
	if !usernamePattern.MatchString(username) {
		return errors.New("username must be 3 to 64 letters, digits, dots, dashes or underscores")
	}
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email {
		return errors.New("invalid email address")
	}
	if len(password) < 12 || len(password) > 72 {
		return errors.New("password must contain between 12 and 72 characters")
	}
	return nil
}
