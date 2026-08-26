package tools

import "errors"

const (
	MinimumPasswordBytes = 12
	MaximumPasswordBytes = 72
)

// ValidatePassword enforces the shared password length policy in bytes.
func ValidatePassword(password string) error {
	if len(password) < MinimumPasswordBytes || len(password) > MaximumPasswordBytes {
		return errors.New("password must contain between 12 and 72 characters")
	}
	return nil
}
