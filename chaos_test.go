package chaos

import (
	"encoding/json"
	"testing"
)

func TestChaosTimeUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "valid UK summer time", input: `"2024-06-15 14:30:00"`, wantErr: false},
		{name: "valid UK winter time", input: `"2024-01-15 00:00:00"`, wantErr: false},
		{name: "invalid format", input: `"not-a-time"`, wantErr: true},
		{name: "RFC3339 format rejected", input: `"2024-01-15T00:00:00Z"`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ct chaosTime
			err := json.Unmarshal([]byte(tt.input), &ct)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAuthForm(t *testing.T) {
	tests := []struct {
		name string
		auth Auth
		keys map[string]string
		// keys that should be absent
		absent []string
	}{
		{
			name: "control auth only",
			auth: Auth{ControlLogin: "user@example.com", ControlPassword: "secret"},
			keys: map[string]string{
				"control_login":    "user@example.com",
				"control_password": "secret",
			},
			absent: []string{"account_number", "account_password"},
		},
		{
			name: "account auth only",
			auth: Auth{AccountNumber: "12345678", AccountPassword: "pass"},
			keys: map[string]string{
				"account_number":   "12345678",
				"account_password": "pass",
			},
			absent: []string{"control_login", "control_password"},
		},
		{
			name: "account with control login",
			auth: Auth{AccountNumber: "12345678", AccountPassword: "pass", ControlLogin: "ctrl@example.com"},
			keys: map[string]string{
				"account_number":   "12345678",
				"account_password": "pass",
				"control_login":    "ctrl@example.com",
			},
			absent: []string{"control_password"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := tt.auth.form()
			for key, want := range tt.keys {
				if got := f.Get(key); got != want {
					t.Errorf("form[%q] = %q, want %q", key, got, want)
				}
			}
			for _, key := range tt.absent {
				if got := f.Get(key); got != "" {
					t.Errorf("form[%q] = %q, want empty", key, got)
				}
			}
		})
	}
}

func TestBroadbandInfoUnmarshal(t *testing.T) {
	const data = `{"info":[{"id":"12345","login":"test@bb.com","postcode":"SW1A1AA","tx_rate":"80000000","rx_rate":"20000000","tx_rate_adjusted":"79000000","quota_monthly":"107374182400","quota_remaining":"53687091200","quota_timestamp":"2024-01-15 00:00:00"}]}`

	r := struct {
		Info  []BroadbandInfo `json:"info"`
		Error string          `json:"error"`
	}{}
	if err := json.Unmarshal([]byte(data), &r); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(r.Info) != 1 {
		t.Fatalf("len(Info) = %d, want 1", len(r.Info))
	}
	line := r.Info[0]
	if line.ID != 12345 {
		t.Errorf("ID = %d, want 12345", line.ID)
	}
	if line.Login != "test@bb.com" {
		t.Errorf("Login = %q, want %q", line.Login, "test@bb.com")
	}
	if line.TXRate != 80000000 {
		t.Errorf("TXRate = %d, want 80000000", line.TXRate)
	}
	if line.RXRate != 20000000 {
		t.Errorf("RXRate = %d, want 20000000", line.RXRate)
	}
	if line.QuotaMonthly != 107374182400 {
		t.Errorf("QuotaMonthly = %d, want 107374182400", line.QuotaMonthly)
	}
	if line.QuotaRemaining != 53687091200 {
		t.Errorf("QuotaRemaining = %d, want 53687091200", line.QuotaRemaining)
	}
}

func TestBroadbandInfoError(t *testing.T) {
	const data = `{"error":"Invalid credentials"}`

	r := struct {
		Info  []BroadbandInfo `json:"info"`
		Error string          `json:"error"`
	}{}
	if err := json.Unmarshal([]byte(data), &r); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if r.Error != "Invalid credentials" {
		t.Errorf("Error = %q, want %q", r.Error, "Invalid credentials")
	}
}

func TestBroadbandQuotaUnmarshal(t *testing.T) {
	const data = `{"quota":[{"id":"12345","quota_monthly":"107374182400","quota_remaining":"53687091200","quota_timestamp":"2024-01-15 00:00:00"}]}`

	r := struct {
		Quota []BroadbandQuota `json:"quota"`
		Error string           `json:"error"`
	}{}
	if err := json.Unmarshal([]byte(data), &r); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(r.Quota) != 1 {
		t.Fatalf("len(Quota) = %d, want 1", len(r.Quota))
	}
	q := r.Quota[0]
	if q.ID != 12345 {
		t.Errorf("ID = %d, want 12345", q.ID)
	}
	if q.QuotaMonthly != 107374182400 {
		t.Errorf("QuotaMonthly = %d, want 107374182400", q.QuotaMonthly)
	}
	if q.QuotaRemaining != 53687091200 {
		t.Errorf("QuotaRemaining = %d, want 53687091200", q.QuotaRemaining)
	}
}
