package validation

import (
	"strings"
	"testing"
)

func TestValidateProxyURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
		errMsg  string
	}{
		// Valid cases
		{
			name:    "valid http proxy with port",
			url:     "http://proxy.example.com:8080",
			wantErr: false,
		},
		{
			name:    "valid https proxy with port",
			url:     "https://proxy.example.com:8080",
			wantErr: false,
		},
		{
			name:    "valid socks5 proxy",
			url:     "socks5://proxy.example.com:1080",
			wantErr: false,
		},
		{
			name:    "valid proxy with auth",
			url:     "http://username:password@proxy.example.com:8080",
			wantErr: false,
		},
		{
			name:    "valid proxy with username only",
			url:     "http://username@proxy.example.com:8080",
			wantErr: false,
		},
		{
			name:    "valid proxy without port (uses default)",
			url:     "http://proxy.example.com",
			wantErr: false,
		},
		{
			name:    "valid proxy with IPv4 address",
			url:     "http://192.168.1.100:8080",
			wantErr: false,
		},
		{
			name:    "valid proxy with IPv6 address",
			url:     "http://[2001:db8::1]:8080",
			wantErr: false,
		},
		{
			name:    "empty URL (optional proxy)",
			url:     "",
			wantErr: false,
		},

		// Invalid cases - missing scheme
		{
			name:    "missing scheme",
			url:     "proxy.example.com:8080",
			wantErr: true,
			errMsg:  "unsupported proxy scheme", // url.Parse treats "proxy.example.com" as scheme
		},

		// Invalid cases - unsupported scheme
		{
			name:    "unsupported scheme ftp",
			url:     "ftp://proxy.example.com:8080",
			wantErr: true,
			errMsg:  "unsupported proxy scheme",
		},
		{
			name:    "unsupported scheme socks4",
			url:     "socks4://proxy.example.com:1080",
			wantErr: true,
			errMsg:  "unsupported proxy scheme",
		},

		// Invalid cases - missing hostname
		{
			name:    "missing hostname",
			url:     "http://:8080",
			wantErr: true,
			errMsg:  "must include hostname",
		},
		{
			name:    "only scheme",
			url:     "http://",
			wantErr: true,
			errMsg:  "must include hostname",
		},

		// Invalid cases - invalid hostname
		{
			name:    "invalid hostname with spaces",
			url:     "http://proxy example.com:8080",
			wantErr: true,
			errMsg:  "invalid proxy URL format",
		},
		{
			name:    "invalid hostname with special chars",
			url:     "http://proxy_example.com:8080",
			wantErr: true,
			errMsg:  "invalid proxy hostname",
		},

		// Invalid cases - invalid port
		{
			name:    "port too high",
			url:     "http://proxy.example.com:99999",
			wantErr: true,
			errMsg:  "invalid proxy port",
		},
		{
			name:    "port zero",
			url:     "http://proxy.example.com:0",
			wantErr: true,
			errMsg:  "invalid proxy port",
		},
		{
			name:    "negative port",
			url:     "http://proxy.example.com:-1",
			wantErr: true,
			errMsg:  "invalid proxy URL format", // url.Parse rejects this early
		},
		{
			name:    "non-numeric port",
			url:     "http://proxy.example.com:abc",
			wantErr: true,
			errMsg:  "invalid proxy URL format", // url.Parse rejects this early
		},

		// Invalid cases - empty username
		{
			name:    "empty username with password",
			url:     "http://:password@proxy.example.com:8080",
			wantErr: true,
			errMsg:  "empty username",
		},

		// Edge cases
		{
			name:    "localhost proxy",
			url:     "http://localhost:8080",
			wantErr: false,
		},
		{
			name:    "proxy with subdomain",
			url:     "http://proxy.internal.example.com:8080",
			wantErr: false,
		},
		{
			name:    "proxy with complex auth",
			url:     "http://user%40example.com:p%40ssw0rd@proxy.example.com:8080",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProxyURL(tt.url)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateProxyURL() expected error, got nil")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateProxyURL() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateProxyURL() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestValidateProxyURL_RealWorldExamples(t *testing.T) {
	// Test real-world proxy configurations
	realWorldTests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "corporate HTTP proxy",
			url:     "http://corpproxy.company.com:8080",
			wantErr: false,
		},
		{
			name:    "squid proxy default port",
			url:     "http://squid.local:3128",
			wantErr: false,
		},
		{
			name:    "SOCKS5 proxy with auth",
			url:     "socks5://user:pass@socks.example.com:1080",
			wantErr: false,
		},
		{
			name:    "HTTPS proxy",
			url:     "https://secure-proxy.example.com:443",
			wantErr: false,
		},
	}

	for _, tt := range realWorldTests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProxyURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateProxyURL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func BenchmarkValidateProxyURL(b *testing.B) {
	urls := []string{
		"http://proxy.example.com:8080",
		"https://proxy.example.com:8080",
		"socks5://user:pass@proxy.example.com:1080",
		"http://192.168.1.100:8080",
		"",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, url := range urls {
			_ = ValidateProxyURL(url)
		}
	}
}
