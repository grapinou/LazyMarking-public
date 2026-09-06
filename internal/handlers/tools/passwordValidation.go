package tools

import "errors"

const (
	MinimumPasswordBytes = 12
	MaximumPasswordBytes = 72
)

// ValidatePassword enforces the shared password length policy in bytes.
func ValidatePassword(password string) error {
	if len(password) < MinimumPasswordBytes || len(password) > MaximumPasswordBytes {
		return errors.New("Le mot de passe doit contenir entre 12 et 72 octets ; certains caractères occupent plusieurs octets.")
	}
	return nil
}
