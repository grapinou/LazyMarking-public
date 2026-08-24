package tools

import (
	"strings"
	"testing"
)

func TestValidateCSVStructure(t *testing.T) {
	tests := []struct {
		name, input string
		wantErr     bool
	}{
		{"valid", "Alex;Martin\nÉlodie;Durand\n", false},
		{"empty", "", true},
		{"missing column", "Alex\n", true},
		{"extra column", "Alex;Martin;Other\n", true},
		{"invalid UTF-8", "Alex;\xff\n", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateCSVStructure(strings.NewReader(tt.input))
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateCSVStructureTruncatesByRune(t *testing.T) {
	input := strings.Repeat("é", 30) + ";Martin\n"
	records, err := ValidateCSVStructure(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if got := len([]rune(records[0][0])); got != 25 {
		t.Fatalf("name length = %d runes, want 25", got)
	}
}
