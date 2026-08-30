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

		// Normalize surrounding whitespace without altering identity punctuation.
		for i := range record {
			if !utf8.ValidString(record[i]) {
				return nil, errors.New("CSV contains invalid UTF-8")
			}
			record[i] = strings.TrimSpace(record[i])
			if record[i] == "" {
				return nil, errors.New("empty file")
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
