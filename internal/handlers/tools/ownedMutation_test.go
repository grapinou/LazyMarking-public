package tools

import (
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleOwnedMutationRows(t *testing.T) {
	tests := []struct {
		name       string
		rows       int64
		wantOK     bool
		wantStatus int
	}{
		{name: "one row succeeds", rows: 1, wantOK: true, wantStatus: http.StatusOK},
		{name: "zero rows is not found", rows: 0, wantStatus: http.StatusNotFound},
		{name: "multiple rows is integrity error", rows: 2, wantStatus: http.StatusInternalServerError},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			if got := HandleOwnedMutationRows(response, tc.rows, "test mutation"); got != tc.wantOK {
				t.Fatalf("HandleOwnedMutationRows() = %v, want %v", got, tc.wantOK)
			}
			if response.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, tc.wantStatus)
			}
		})
	}
}

func TestHandleOwnedLookupError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{name: "missing or foreign row", err: sql.ErrNoRows, wantStatus: http.StatusNotFound},
		{name: "wrapped missing row", err: errors.Join(errors.New("lookup"), sql.ErrNoRows), wantStatus: http.StatusNotFound},
		{name: "database failure", err: errors.New("database unavailable"), wantStatus: http.StatusInternalServerError},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			HandleOwnedLookupError(response, tc.err, "test lookup")
			if response.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, tc.wantStatus)
			}
		})
	}
}
