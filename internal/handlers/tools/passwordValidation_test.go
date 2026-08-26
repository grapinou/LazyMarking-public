package tools

import (
	"strings"
	"testing"
)

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{name: "empty", password: "", wantErr: true},
		{name: "eleven bytes", password: "12345678901", wantErr: true},
		{name: "twelve bytes", password: "123456789012"},
		{name: "seventy two bytes", password: strings.Repeat("a", 72)},
		{name: "seventy three bytes", password: strings.Repeat("a", 73), wantErr: true},
		{name: "unicode twelve bytes six runes", password: "éééééé"},
		{name: "unicode ten bytes five runes", password: "ééééé", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidatePassword(tc.password)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ValidatePassword(%q) error = %v, wantErr %v; byte length=%d", tc.password, err, tc.wantErr, len(tc.password))
			}
		})
	}
}
