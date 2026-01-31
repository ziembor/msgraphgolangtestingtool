package main

import (
	"testing"
)

func TestValidateConfiguration_Action(t *testing.T) {
	tests := []struct {
		name    string
		action  string
		wantErr bool
	}{
		{"valid testconnect", "testconnect", false},
		{"uppercase TESTCONNECT", "TESTCONNECT", true},
		{"invalid action", "invalid", true},
		{"empty action", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Action: tt.action,
				Host:   "imap.example.com",
				Port:   143,
			}
			err := validateConfiguration(config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfiguration() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateConfiguration_Host(t *testing.T) {
	tests := []struct {
		name    string
		host    string
		wantErr bool
	}{
		{"valid host", "imap.example.com", false},
		{"empty host", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Action: "testconnect",
				Host:   tt.host,
				Port:   143,
			}
			err := validateConfiguration(config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfiguration() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateConfiguration_Port(t *testing.T) {
	tests := []struct {
		name    string
		port    int
		wantErr bool
	}{
		{"valid port 143", 143, false},
		{"valid port 993", 993, false},
		{"port 0", 0, true},
		{"port negative", -1, true},
		{"port too high", 70000, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Action: "testconnect",
				Host:   "imap.example.com",
				Port:   tt.port,
			}
			err := validateConfiguration(config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfiguration() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateConfiguration_TLS(t *testing.T) {
	tests := []struct {
		name     string
		imaps    bool
		startTLS bool
		wantErr  bool
	}{
		{"no TLS", false, false, false},
		{"IMAPS only", true, false, false},
		{"STARTTLS only", false, true, false},
		{"both IMAPS and STARTTLS", true, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Action:   "testconnect",
				Host:     "imap.example.com",
				Port:     143,
				IMAPS:    tt.imaps,
				StartTLS: tt.startTLS,
			}
			err := validateConfiguration(config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfiguration() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
