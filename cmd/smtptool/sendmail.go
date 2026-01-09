package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"msgraphgolangtestingtool/internal/common/logger"
	smtptls "msgraphgolangtestingtool/internal/smtp/tls"
)

// sendMail performs end-to-end email sending test.
func sendMail(ctx context.Context, config *Config, csvLogger *logger.CSVLogger, slogLogger *slog.Logger) error {
	fmt.Printf("Sending test email via %s:%d...\n\n", config.Host, config.Port)

	// Write CSV header
	if shouldWrite, _ := csvLogger.ShouldWriteHeader(); shouldWrite {
		csvLogger.WriteHeader([]string{
			"Action", "Status", "Server", "Port", "From", "To",
			"Subject", "SMTP_Response_Code", "Message_ID", "Error",
		})
	}

	fmt.Printf("From:    %s\n", config.From)
	fmt.Printf("To:      %s\n", strings.Join(config.To, ", "))
	fmt.Printf("Subject: %s\n\n", config.Subject)

	// Create and connect client
	client := NewSMTPClient(config.Host, config.Port, config)
	logger.LogDebug(slogLogger, "Connecting to SMTP server")

	if err := client.Connect(ctx); err != nil {
		logger.LogError(slogLogger, "Connection failed", "error", err)
		csvLogger.WriteRow([]string{
			config.Action, "FAILURE", config.Host, fmt.Sprintf("%d", config.Port),
			config.From, strings.Join(config.To, ", "), config.Subject, "", "", err.Error(),
		})
		return err
	}
	defer client.Close()

	fmt.Printf("✓ Connected\n")

	// Send EHLO
	logger.LogDebug(slogLogger, "Sending EHLO command")
	caps, err := client.EHLO("smtptool.local")
	if err != nil {
		logger.LogError(slogLogger, "EHLO failed", "error", err)
		csvLogger.WriteRow([]string{
			config.Action, "FAILURE", config.Host, fmt.Sprintf("%d", config.Port),
			config.From, strings.Join(config.To, ", "), config.Subject, "", "", err.Error(),
		})
		return err
	}

	// STARTTLS if on port 25/587 and available
	if (config.Port == 25 || config.Port == 587) && caps.SupportsSTARTTLS() {
		fmt.Println("Upgrading to TLS...")
		tlsConfig := &tls.Config{
			ServerName:         config.Host,
			InsecureSkipVerify: config.SkipVerify,
			MinVersion:         smtptls.ParseTLSVersion(config.TLSVersion),
		}

		_, err := client.StartTLS(tlsConfig)
		if err != nil {
			logger.LogError(slogLogger, "STARTTLS failed", "error", err)
			csvLogger.WriteRow([]string{
				config.Action, "FAILURE", config.Host, fmt.Sprintf("%d", config.Port),
				config.From, strings.Join(config.To, ", "), config.Subject, "", "", fmt.Sprintf("STARTTLS failed: %v", err),
			})
			return fmt.Errorf("STARTTLS failed: %w", err)
		}

		fmt.Println("✓ TLS upgrade successful")

		// Re-run EHLO on encrypted connection
		caps, err = client.EHLO("smtptool.local")
		if err != nil {
			return fmt.Errorf("EHLO on encrypted connection failed: %w", err)
		}
	}

	// Authenticate if credentials provided
	if config.Username != "" && config.Password != "" {
		fmt.Println("Authenticating...")
		authMechanisms := caps.GetAuthMechanisms()
		methodToUse := selectAuthMechanism([]string{config.AuthMethod}, authMechanisms)

		if methodToUse == "" {
			msg := "No compatible authentication mechanism found"
			csvLogger.WriteRow([]string{
				config.Action, "FAILURE", config.Host, fmt.Sprintf("%d", config.Port),
				config.From, strings.Join(config.To, ", "), config.Subject, "", "", msg,
			})
			return fmt.Errorf(msg)
		}

		if err := client.Auth(config.Username, config.Password, []string{methodToUse}); err != nil {
			logger.LogError(slogLogger, "Authentication failed", "error", err)
			csvLogger.WriteRow([]string{
				config.Action, "FAILURE", config.Host, fmt.Sprintf("%d", config.Port),
				config.From, strings.Join(config.To, ", "), config.Subject, "", "", fmt.Sprintf("Auth failed: %v", err),
			})
			return fmt.Errorf("authentication failed: %w", err)
		}

		fmt.Println("✓ Authentication successful")
	}

	// Build email message
	messageData := buildEmailMessage(config.From, config.To, config.Subject, config.Body)
	messageID := generateMessageID(config.Host)

	// Send email
	fmt.Println("\nSending message...")
	logger.LogDebug(slogLogger, "Sending email", "from", config.From, "to", config.To)

	err = client.SendMail(config.From, config.To, messageData)
	if err != nil {
		logger.LogError(slogLogger, "Failed to send email", "error", err)
		csvLogger.WriteRow([]string{
			config.Action, "FAILURE", config.Host, fmt.Sprintf("%d", config.Port),
			config.From, strings.Join(config.To, ", "), config.Subject, "", "", err.Error(),
		})
		return fmt.Errorf("failed to send email: %w", err)
	}

	fmt.Println("✓ Message sent successfully")
	fmt.Printf("  Message-ID: <%s>\n", messageID)

	// Log to CSV
	csvLogger.WriteRow([]string{
		config.Action, "SUCCESS", config.Host, fmt.Sprintf("%d", config.Port),
		config.From, strings.Join(config.To, ", "), config.Subject,
		"250", messageID, "",
	})

	fmt.Println("\n✓ Email sending test completed successfully")
	logger.LogInfo(slogLogger, "sendmail completed successfully", "messageID", messageID)

	return nil
}

// buildEmailMessage constructs an RFC 5322 email message.
func buildEmailMessage(from string, to []string, subject, body string) []byte {
	messageID := generateMessageID("")
	date := time.Now().Format(time.RFC1123Z)

	message := fmt.Sprintf("Message-ID: <%s>\r\n", messageID)
	message += fmt.Sprintf("Date: %s\r\n", date)
	message += fmt.Sprintf("From: %s\r\n", from)
	message += fmt.Sprintf("To: %s\r\n", strings.Join(to, ", "))
	message += fmt.Sprintf("Subject: %s\r\n", subject)
	message += "MIME-Version: 1.0\r\n"
	message += "Content-Type: text/plain; charset=UTF-8\r\n"
	message += "\r\n"
	message += body
	message += "\r\n"

	return []byte(message)
}

// generateMessageID creates a unique message ID.
func generateMessageID(host string) string {
	timestamp := time.Now().UnixNano()
	if host == "" {
		host = "smtptool"
	}
	return fmt.Sprintf("%d.smtptool@%s", timestamp, host)
}
