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

func TestValidateCSVStructurePreservesLongNames(t *testing.T) {
	firstName := "Jean-Christophe-Alexandre"
	lastName := "Dupond-Dupont-Très-Long"
	input := firstName + ";" + lastName + "\n"
	records, err := ValidateCSVStructure(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if records[0][0] != firstName || records[0][1] != lastName {
		t.Fatalf("record=%q, want [%q %q]", records[0], firstName, lastName)
	}
}

func TestValidateCSVStructurePreservesLongUnicodeNamesExactly(t *testing.T) {
	firstName := "Éléonore-Alexandrine-Çağdaş-李小龍"
	lastName := "D’Estaing-Coëffé-Ångström-非常に長い名前"
	input := "  " + firstName + "  ;  " + lastName + "  \n"
	records, err := ValidateCSVStructure(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if records[0][0] != firstName || records[0][1] != lastName {
		t.Fatalf("record=%q, want [%q %q]", records[0], firstName, lastName)
	}
	if len([]rune(firstName)) <= 25 || len([]rune(lastName)) <= 25 {
		t.Fatal("test names must exceed the former 25-rune limit")
	}
}
