package tools

import (
	"encoding/csv"
	"errors"
	"io"
	"strings"
)

// Validation de structure (2 colonnes "prénom" ; "nom")
func ValidateCSVStructure(reader io.Reader) ([][]string, error) {
	csvReader := csv.NewReader(reader)
	csvReader.Comma = ';'
	csvReader.LazyQuotes = true

	var records [][]string
	line := 0
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		line++

		// Vérifie nombre de colonnes
		if len(record) != 2 {
			return nil, errors.New("invalid structure, more than 2 columns")
		}

		// Trim et vérifie contenu
		for i := range record {
			record[i] = strings.Trim(record[i], "\" ")
			if record[i] == "" {
				return nil, errors.New("empty file")
			}

			const maxNameLength = 25

			if len(record[i]) > maxNameLength {
				record[i] = record[i][:maxNameLength]
			}
		}

		records = append(records, record)
	}

	return records, nil
}
