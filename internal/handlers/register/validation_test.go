package register

import "testing"

func TestValidateRegistration(t *testing.T) {
	tests := []struct {
		name, username, email, password string
		wantErr                         bool
	}{
		{"valid", "alice.dupont", "alice@example.com", "a-long-password", false},
		{"path traversal", "../alice", "alice@example.com", "a-long-password", true},
		{"invalid email", "alice", "not-an-email", "a-long-password", true},
		{"short password", "alice", "alice@example.com", "short", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRegistration(tt.username, tt.email, tt.password)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateRegistration() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
