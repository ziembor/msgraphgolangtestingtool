package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"msgraphgolangtestingtool/internal/common/logger"
	"msgraphgolangtestingtool/internal/smtp/exchange"
)

// testConnect performs basic SMTP connectivity and capability testing.
func testConnect(ctx context.Context, config *Config, csvLogger *logger.CSVLogger, slogLogger *slog.Logger) error {
	fmt.Printf("Testing SMTP connectivity to %s:%d...\n\n", config.Host, config.Port)

	// Write CSV header
	if shouldWrite, _ := csvLogger.ShouldWriteHeader(); shouldWrite {
		csvLogger.WriteHeader([]string{"Action", "Status", "Server", "Port", "Connected", "Banner", "Capabilities", "Exchange_Detected", "Error"})
	}

	// Create client
	client := NewSMTPClient(config.Host, config.Port, config)

	// Connect
	logger.LogDebug(slogLogger, "Connecting to SMTP server", "host", config.Host, "port", config.Port)
	if err := client.Connect(ctx); err != nil {
		logger.LogError(slogLogger, "Connection failed", "error", err)
		csvLogger.WriteRow([]string{
			config.Action, "FAILURE", config.Host,
			fmt.Sprintf("%d", config.Port), "false", "", "", "false", err.Error(),
		})
		return err
	}
	defer client.Close()

	fmt.Printf("✓ Connected successfully\n")
	fmt.Printf("  Banner: %s\n\n", client.GetBanner())

	// Send EHLO
	logger.LogDebug(slogLogger, "Sending EHLO command")
	caps, err := client.EHLO("smtptool.local")
	if err != nil {
		logger.LogError(slogLogger, "EHLO failed", "error", err)
		csvLogger.WriteRow([]string{
			config.Action, "FAILURE", config.Host,
			fmt.Sprintf("%d", config.Port), "true", client.GetBanner(), "", "false", err.Error(),
		})
		return err
	}

	// Display capabilities
	fmt.Println("Server Capabilities:")
	for cap, params := range caps {
		if len(params) > 0 {
			fmt.Printf("  • %s: %s\n", cap, strings.Join(params, ", "))
		} else {
			fmt.Printf("  • %s\n", cap)
		}
	}
	fmt.Println()

	// Detect Exchange
	exchangeInfo := exchange.DetectExchange(client.GetBanner(), caps)
	if exchangeInfo.IsExchange {
		fmt.Print(exchange.FormatExchangeInfo(exchangeInfo, caps))
	}

	// Log to CSV
	capsStr := caps.String()
	csvLogger.WriteRow([]string{
		config.Action, "SUCCESS", config.Host,
		fmt.Sprintf("%d", config.Port), "true", client.GetBanner(),
		capsStr, fmt.Sprintf("%t", exchangeInfo.IsExchange), "",
	})

	fmt.Println("✓ Connectivity test completed successfully")
	logger.LogInfo(slogLogger, "testconnect completed successfully")

	return nil
}
