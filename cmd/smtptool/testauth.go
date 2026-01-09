package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"msgraphgolangtestingtool/internal/common/logger"
	smtptls "msgraphgolangtestingtool/internal/smtp/tls"
)

// testAuth performs SMTP authentication testing.
func testAuth(ctx context.Context, config *Config, csvLogger *logger.CSVLogger, slogLogger *slog.Logger) error {
	fmt.Printf("Testing SMTP authentication on %s:%d...\n\n", config.Host, config.Port)

	// Write CSV header
	if shouldWrite, _ := csvLogger.ShouldWriteHeader(); shouldWrite {
		csvLogger.WriteHeader([]string{
			"Action", "Status", "Server", "Port", "Username",
			"Auth_Mechanisms_Available", "Auth_Method_Used", "Auth_Result", "Error",
		})
	}

	// Create and connect client
	client := NewSMTPClient(config.Host, config.Port, config)
	logger.LogDebug(slogLogger, "Connecting to SMTP server")

	if err := client.Connect(ctx); err != nil {
		logger.LogError(slogLogger, "Connection failed", "error", err)
		csvLogger.WriteRow([]string{
			config.Action, "FAILURE", config.Host, fmt.Sprintf("%d", config.Port),
			config.Username, "", "", "FAILURE", err.Error(),
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
			config.Username, "", "", "FAILURE", err.Error(),
		})
		return err
	}

	// Check for AUTH capability
	authMechanisms := caps.GetAuthMechanisms()
	if len(authMechanisms) == 0 {
		msg := "Server does not advertise AUTH capability"
		fmt.Printf("✗ %s\n", msg)
		logger.LogWarn(slogLogger, msg)
		csvLogger.WriteRow([]string{
			config.Action, "FAILURE", config.Host, fmt.Sprintf("%d", config.Port),
			config.Username, "none", "", "FAILURE", msg,
		})
		return errors.New(msg)
	}

	fmt.Printf("✓ Server supports AUTH mechanisms: %s\n\n", strings.Join(authMechanisms, ", "))

	// STARTTLS if on port 25/587 and available
	if (config.Port == 25 || config.Port == 587) && caps.SupportsSTARTTLS() {
		fmt.Println("Upgrading to TLS before authentication...")
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
				config.Username, strings.Join(authMechanisms, ", "), "", "FAILURE", fmt.Sprintf("STARTTLS failed: %v", err),
			})
			return fmt.Errorf("STARTTLS failed: %w", err)
		}

		fmt.Println("✓ TLS upgrade successful")

		// Re-run EHLO on encrypted connection
		caps, err = client.EHLO("smtptool.local")
		if err != nil {
			return fmt.Errorf("EHLO on encrypted connection failed: %w", err)
		}
		authMechanisms = caps.GetAuthMechanisms()
	}

	// Determine auth method to use
	var methodsToTry []string
	if config.AuthMethod == "auto" {
		methodsToTry = []string{"auto"}
	} else {
		methodsToTry = []string{config.AuthMethod}
	}

	methodUsed := selectAuthMechanism(methodsToTry, authMechanisms)
	if methodUsed == "" {
		msg := fmt.Sprintf("No compatible authentication mechanism found (requested: %s, available: %s)",
			config.AuthMethod, strings.Join(authMechanisms, ", "))
		fmt.Printf("✗ %s\n", msg)
		csvLogger.WriteRow([]string{
			config.Action, "FAILURE", config.Host, fmt.Sprintf("%d", config.Port),
			config.Username, strings.Join(authMechanisms, ", "), "", "FAILURE", msg,
		})
		return errors.New(msg)
	}

	fmt.Printf("Attempting authentication with method: %s\n", methodUsed)
	logger.LogDebug(slogLogger, "Authenticating", "method", methodUsed, "username", config.Username)

	// Authenticate
	err = client.Auth(config.Username, config.Password, []string{methodUsed})

	authResult := "SUCCESS"
	status := "SUCCESS"
	errorMsg := ""

	if err != nil {
		authResult = "FAILURE"
		status = "FAILURE"
		errorMsg = err.Error()
		fmt.Printf("\n✗ Authentication failed: %v\n", err)
		logger.LogError(slogLogger, "Authentication failed", "error", err)
	} else {
		fmt.Printf("\n✓ Authentication successful\n")
		logger.LogInfo(slogLogger, "Authentication successful", "method", methodUsed)
	}

	// Log to CSV
	csvLogger.WriteRow([]string{
		config.Action, status, config.Host, fmt.Sprintf("%d", config.Port),
		config.Username, strings.Join(authMechanisms, ", "),
		methodUsed, authResult, errorMsg,
	})

	if err != nil {
		return err
	}

	fmt.Println("\n✓ Authentication test completed successfully")
	return nil
}
