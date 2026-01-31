package main

import (
	"testing"
)

func TestMaskUsername(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"user@example.com", "us****om"},
		{"ab@example.com", "ab****om"},
		{"test", "****"},
		{"ab", "****"},
		{"a", "****"},
	}

	for _, tt := range tests {
		result := maskUsername(tt.input)
		if result != tt.expected {
			t.Errorf("maskUsername(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestMaskPassword(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"secretpassword", "se****rd"},
		{"password", "pa****rd"},
		{"test", "****"},
		{"ab", "****"},
		{"", ""},
	}

	for _, tt := range tests {
		result := maskPassword(tt.input)
		if result != tt.expected {
			t.Errorf("maskPassword(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestMaskAccessToken(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ya29.a0ARrdaM_1234567890abcdefghij", "ya29.a0A...ghij"},
		{"short", "sh...ort"},
		{"1234", "12...34"},
		{"abc", "a...bc"},
		{"ab", "a...b"},
		{"a", "...a"},
		{"", ""},
	}

	for _, tt := range tests {
		result := maskAccessToken(tt.input)
		if result != tt.expected {
			t.Errorf("maskAccessToken(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
