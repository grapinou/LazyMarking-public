package tools

import (
	"bytes"
	"encoding/csv"
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

func TestValidateCSVStructurePreservesLiteralQuotesFromRealCSV(t *testing.T) {
	tests := []struct {
		name      string
		firstName string
		lastName  string
	}{
		{"quotes inside", `Jean "Junior"`, "Martin"},
		{"leading quote", `"Jean`, "Martin"},
		{"trailing quote", `Jean"`, "Martin"},
		{"quotes at both ends", `"Jean"`, "Martin"},
		{"surrounding spaces only", `  Jean "Junior"  `, `  D'Arc  `},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var input bytes.Buffer
			writer := csv.NewWriter(&input)
			writer.Comma = ';'
			if err := writer.Write([]string{test.firstName, test.lastName}); err != nil {
				t.Fatal(err)
			}
			writer.Flush()
			if err := writer.Error(); err != nil {
				t.Fatal(err)
			}

			records, err := ValidateCSVStructure(&input)
			if err != nil {
				t.Fatal(err)
			}
			wantFirstName := strings.TrimSpace(test.firstName)
			wantLastName := strings.TrimSpace(test.lastName)
			if records[0][0] != wantFirstName || records[0][1] != wantLastName {
				t.Fatalf("record=%q, want [%q %q]", records[0], wantFirstName, wantLastName)
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
