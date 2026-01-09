package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"msgraphgolangtestingtool/internal/common/logger"
	"msgraphgolangtestingtool/internal/common/version"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Setup signal handling for graceful shutdown
	ctx, cancel := setupSignalHandling()
	defer cancel()

	// Parse configuration
	config := parseAndConfigureFlags()

	// Handle version flag
	if config.ShowVersion {
		fmt.Printf("SMTP Connectivity Testing Tool - Version %s\n", version.Get())
		fmt.Println("Part of msgraphgolangtestingtool suite")
		fmt.Println("Repository: https://github.com/ziembor/msgraphgolangtestingtool")
		return nil
	}

	// Validate configuration
	if err := validateConfiguration(config); err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	// Setup structured logger
	slogLogger := logger.SetupLogger(config.VerboseMode, config.LogLevel)
	logger.LogInfo(slogLogger, "SMTP Connectivity Testing Tool started", "action", config.Action, "host", config.Host, "port", config.Port)

	// Initialize CSV logger
	csvLogger, err := logger.NewCSVLogger("smtptool", config.Action)
	if err != nil {
		return fmt.Errorf("failed to initialize CSV logger: %w", err)
	}
	defer csvLogger.Close()

	// Execute the action
	if err := executeAction(ctx, config, csvLogger, slogLogger); err != nil {
		logger.LogError(slogLogger, "Action failed", "error", err)
		return err
	}

	logger.LogInfo(slogLogger, "Action completed successfully")
	return nil
}

// setupSignalHandling sets up graceful shutdown on SIGINT/SIGTERM.
func setupSignalHandling() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\n\nReceived interrupt signal. Shutting down gracefully...")
		cancel()
	}()

	return ctx, cancel
}
