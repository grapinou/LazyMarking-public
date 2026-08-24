package tools

import (
	"encoding/csv"
	"errors"
	"io"
	"strings"
	"unicode/utf8"
)

// Validation de structure (2 colonnes "prénom" ; "nom")
func ValidateCSVStructure(reader io.Reader) ([][]string, error) {
	csvReader := csv.NewReader(reader)
	csvReader.Comma = ';'
	csvReader.LazyQuotes = false
	csvReader.FieldsPerRecord = 2

	var records [][]string
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		// Vérifie nombre de colonnes
		if len(record) != 2 {
			return nil, errors.New("invalid structure, more than 2 columns")
		}

		// Trim et vérifie contenu
		for i := range record {
			if !utf8.ValidString(record[i]) {
				return nil, errors.New("CSV contains invalid UTF-8")
			}
			record[i] = strings.Trim(record[i], "\" ")
			if record[i] == "" {
				return nil, errors.New("empty file")
			}

			const maxNameLength = 25

			runes := []rune(record[i])
			if len(runes) > maxNameLength {
				record[i] = string(runes[:maxNameLength])
			}
		}

		records = append(records, record)
		if len(records) > 10000 {
			return nil, errors.New("CSV contains too many records")
		}
	}
	if len(records) == 0 {
		return nil, errors.New("CSV is empty")
	}
	return records, nil
}
