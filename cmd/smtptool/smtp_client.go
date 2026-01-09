package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"net/textproto"
	"strings"

	"msgraphgolangtestingtool/internal/smtp/protocol"
)

// SMTPClient wraps SMTP connection with enhanced diagnostics.
type SMTPClient struct {
	conn       net.Conn
	reader     *bufio.Reader
	host       string
	port       int
	config     *Config
	banner     string
	capabilities protocol.Capabilities
}

// NewSMTPClient creates a new SMTP client.
func NewSMTPClient(host string, port int, config *Config) *SMTPClient {
	return &SMTPClient{
		host:   host,
		port:   port,
		config: config,
	}
}

// Connect establishes a TCP connection and reads the banner.
func (c *SMTPClient) Connect(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", c.host, c.port)

	// Use context-aware dialer
	dialer := &net.Dialer{
		Timeout: c.config.Timeout,
	}

	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	c.conn = conn
	c.reader = bufio.NewReader(conn)

	// Read banner (220 response) with timeout
	resp, err := protocol.ReadResponseWithTimeout(c.reader, protocol.DefaultResponseTimeout)
	if err != nil {
		c.conn.Close()
		return fmt.Errorf("failed to read banner: %w", err)
	}

	if !resp.IsSuccess() {
		c.conn.Close()
		return fmt.Errorf("unexpected banner response: %d %s", resp.Code, resp.Message)
	}

	c.banner = resp.Message

	return nil
}

// EHLO sends EHLO command and parses capabilities.
func (c *SMTPClient) EHLO(hostname string) (protocol.Capabilities, error) {
	// Send EHLO command
	cmd := protocol.EHLO(hostname)
	if _, err := c.conn.Write([]byte(cmd)); err != nil {
		return nil, fmt.Errorf("failed to send EHLO: %w", err)
	}

	// Read response with timeout
	resp, err := protocol.ReadResponseWithTimeout(c.reader, protocol.DefaultResponseTimeout)
	if err != nil {
		return nil, fmt.Errorf("failed to read EHLO response: %w", err)
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("EHLO failed: %d %s", resp.Code, resp.Message)
	}

	// Parse capabilities
	c.capabilities = protocol.ParseCapabilities(resp.Lines)

	return c.capabilities, nil
}

// StartTLS upgrades the connection to TLS.
func (c *SMTPClient) StartTLS(tlsConfig *tls.Config) (*tls.ConnectionState, error) {
	// Send STARTTLS command
	cmd := protocol.STARTTLS()
	if _, err := c.conn.Write([]byte(cmd)); err != nil {
		return nil, fmt.Errorf("failed to send STARTTLS: %w", err)
	}

	// Read response (expect 220) with timeout
	resp, err := protocol.ReadResponseWithTimeout(c.reader, protocol.DefaultResponseTimeout)
	if err != nil {
		return nil, fmt.Errorf("failed to read STARTTLS response: %w", err)
	}

	if resp.Code != 220 {
		return nil, fmt.Errorf("STARTTLS failed: %d %s", resp.Code, resp.Message)
	}

	// Perform TLS handshake
	tlsConn := tls.Client(c.conn, tlsConfig)
	if err := tlsConn.HandshakeContext(context.Background()); err != nil {
		return nil, fmt.Errorf("TLS handshake failed: %w", err)
	}

	// Update connection and reader
	c.conn = tlsConn
	c.reader = bufio.NewReader(tlsConn)

	// Get connection state
	state := tlsConn.ConnectionState()

	return &state, nil
}

// Auth performs SMTP authentication.
func (c *SMTPClient) Auth(username, password string, mechanisms []string) error {
	// Determine which mechanism to use
	mechanism := selectAuthMechanism(mechanisms, c.capabilities.GetAuthMechanisms())
	if mechanism == "" {
		return fmt.Errorf("no compatible authentication mechanism found")
	}

	// Create appropriate auth
	var auth smtp.Auth
	switch mechanism {
	case "PLAIN":
		auth = smtp.PlainAuth("", username, password, c.host)
	case "LOGIN":
		auth = &loginAuth{username, password}
	case "CRAM-MD5":
		auth = smtp.CRAMMD5Auth(username, password)
	default:
		return fmt.Errorf("unsupported authentication mechanism: %s", mechanism)
	}

	// Create temporary SMTP client for auth
	smtpClient := &smtp.Client{Text: textproto.NewConn(c.conn)}
	if err := smtpClient.Auth(auth); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	return nil
}

// SendMail sends an email message.
func (c *SMTPClient) SendMail(from string, to []string, data []byte) error {
	// Use stdlib smtp client for mail sending
	smtpClient := &smtp.Client{Text: textproto.NewConn(c.conn)}

	// MAIL FROM
	if err := smtpClient.Mail(from); err != nil {
		return fmt.Errorf("MAIL FROM failed: %w", err)
	}

	// RCPT TO
	for _, recipient := range to {
		if err := smtpClient.Rcpt(recipient); err != nil {
			return fmt.Errorf("RCPT TO failed for %s: %w", recipient, err)
		}
	}

	// DATA
	w, err := smtpClient.Data()
	if err != nil {
		return fmt.Errorf("DATA command failed: %w", err)
	}

	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close DATA: %w", err)
	}

	return nil
}

// Close closes the connection.
func (c *SMTPClient) Close() error {
	if c.conn != nil {
		// Send QUIT
		c.conn.Write([]byte(protocol.QUIT()))
		return c.conn.Close()
	}
	return nil
}

// GetBanner returns the server banner.
func (c *SMTPClient) GetBanner() string {
	return c.banner
}

// GetCapabilities returns the server capabilities.
func (c *SMTPClient) GetCapabilities() protocol.Capabilities {
	return c.capabilities
}

// selectAuthMechanism selects the best authentication mechanism.
func selectAuthMechanism(requested []string, available []string) string {
	// If specific mechanism requested
	if len(requested) > 0 && requested[0] != "auto" {
		for _, req := range requested {
			for _, avail := range available {
				if strings.EqualFold(req, avail) {
					return strings.ToUpper(req)
				}
			}
		}
		return ""
	}

	// Auto-select: prefer stronger mechanisms
	preferenceOrder := []string{"CRAM-MD5", "PLAIN", "LOGIN"}
	for _, preferred := range preferenceOrder {
		for _, avail := range available {
			if strings.EqualFold(preferred, avail) {
				return preferred
			}
		}
	}

	return ""
}

// loginAuth implements LOGIN authentication.
type loginAuth struct {
	username string
	password string
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", nil, nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		prompt := strings.ToLower(string(fromServer))
		if strings.Contains(prompt, "username") {
			return []byte(a.username), nil
		} else if strings.Contains(prompt, "password") {
			return []byte(a.password), nil
		}
	}
	return nil, nil
}
