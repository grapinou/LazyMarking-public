package tools

import "errors"

const (
	MinimumPasswordBytes = 12
	MaximumPasswordBytes = 72
)

// ValidatePassword enforces the shared password length policy in bytes.
func ValidatePassword(password string) error {
	if len(password) < MinimumPasswordBytes || len(password) > MaximumPasswordBytes {
		return errors.New("La longueur du mot de passe est hors des limites acceptées. Avec des lettres non accentuées, chiffres et signes courants : de 12 à 72 caractères. Les accents et émojis comptent davantage dans cette limite.")
	}
	return nil
}
