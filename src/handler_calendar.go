package main

import (
	"context"
	"fmt"
	"log"
	"time"

	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/microsoftgraph/msgraph-sdk-go/users"
)

// listEvents retrieves and displays upcoming calendar events for a user.
// Supports both text and JSON output formats.
func listEvents(ctx context.Context, client *msgraphsdk.GraphServiceClient, mailbox string, count int, config *Config, logger *CSVLogger) error {
	// Configure request to get top N events
	requestConfig := &users.ItemEventsRequestBuilderGetRequestConfiguration{
		QueryParameters: &users.ItemEventsRequestBuilderGetQueryParameters{
			Top: Int32Ptr(int32(count)),
		},
	}

	logVerbose(config.VerboseMode, "Calling Graph API: GET /users/%s/events?$top=%d", mailbox, count)

	// Execute API call with retry logic
	var getValueFunc func() []models.Eventable
	err := retryWithBackoff(ctx, config.MaxRetries, config.RetryDelay, func() error {
		apiResult, apiErr := client.Users().ByUserId(mailbox).Events().Get(ctx, requestConfig)
		if apiErr == nil {
			getValueFunc = apiResult.GetValue
		}
		return apiErr
	})

	if err != nil {
		// Enrich error with rate limit and service error details
		enrichedErr := enrichGraphAPIError(err, logger, "listEvents")
		return fmt.Errorf("error fetching calendar for %s: %w", mailbox, enrichedErr)
	}

	events := getValueFunc()
	eventCount := len(events)

	logVerbose(config.VerboseMode, "API response received: %d events", eventCount)

	if config.OutputFormat == "json" {
		printJSON(formatEventsOutput(events))
	} else {
		fmt.Printf("Upcoming events for %s:\n", mailbox)

		if eventCount == 0 {
			fmt.Println("No events found.")
		} else {
			for _, event := range events {
				subject := "N/A"
				if event.GetSubject() != nil {
					subject = *event.GetSubject()
				}

				id := "N/A"
				if event.GetId() != nil {
					id = *event.GetId()
				}

				fmt.Printf("- %s (ID: %s)\n", subject, id)
			}
			// Log summary entry after all events
			fmt.Printf("\nTotal events retrieved: %d\n", eventCount)
		}
	}

	// Always write to CSV logger regardless of output format
	if logger != nil {
		if eventCount == 0 {
			logger.WriteRow([]string{ActionGetEvents, StatusSuccess, mailbox, "No events found (0 events)", "N/A"})
		} else {
			for _, event := range events {
				subject := "N/A"
				if event.GetSubject() != nil {
					subject = *event.GetSubject()
				}
				id := "N/A"
				if event.GetId() != nil {
					id = *event.GetId()
				}
				logger.WriteRow([]string{ActionGetEvents, StatusSuccess, mailbox, subject, id})
			}
			logger.WriteRow([]string{ActionGetEvents, StatusSuccess, mailbox, fmt.Sprintf("Retrieved %d event(s)", eventCount), "SUMMARY"})
		}
	}

	return nil
}

// createInvite creates a calendar event/invitation via Microsoft Graph API.
// Supports time parsing in multiple formats and includes WhatIf dry run mode.
func createInvite(ctx context.Context, client *msgraphsdk.GraphServiceClient, mailbox, subject, startTimeStr, endTimeStr string, config *Config, logger *CSVLogger) {
	event := models.NewEvent()
	event.SetSubject(&subject)

	// Parse start time, default to now if not provided
	var startTime time.Time
	var err error
	if startTimeStr == "" {
		startTime = time.Now()
	} else {
		startTime, err = parseFlexibleTime(startTimeStr)
		if err != nil {
			log.Printf("Error parsing start time: %v. Using current time instead.", err)
			startTime = time.Now()
		}
	}

	// Parse end time, default to 1 hour after start if not provided
	var endTime time.Time
	if endTimeStr == "" {
		endTime = startTime.Add(1 * time.Hour)
	} else {
		endTime, err = parseFlexibleTime(endTimeStr)
		if err != nil {
			log.Printf("Error parsing end time: %v. Using start + 1 hour instead.", err)
			endTime = startTime.Add(1 * time.Hour)
		}
	}

	// Set start time
	startDateTime := models.NewDateTimeTimeZone()
	startTimeFormatted := startTime.Format(time.RFC3339)
	startDateTime.SetDateTime(&startTimeFormatted)
	timezone := "UTC"
	startDateTime.SetTimeZone(&timezone)
	event.SetStart(startDateTime)

	// Set end time
	endDateTime := models.NewDateTimeTimeZone()
	endTimeFormatted := endTime.Format(time.RFC3339)
	endDateTime.SetDateTime(&endTimeFormatted)
	endDateTime.SetTimeZone(&timezone)
	event.SetEnd(endDateTime)

	status := StatusSuccess
	eventID := "N/A"

	// WhatIf / Dry Run Mode
	if config.WhatIf {
		duration := endTime.Sub(startTime)
		fmt.Println("========================================")
		fmt.Println("WHATIF MODE - DRY RUN (Calendar invite NOT created)")
		fmt.Println("========================================")
		fmt.Printf("Mailbox: %s\n", mailbox)
		fmt.Printf("Subject: %s\n", subject)
		fmt.Printf("Start Time: %s\n", startTime.Format("2006-01-02 15:04:05 MST"))
		fmt.Printf("End Time: %s\n", endTime.Format("2006-01-02 15:04:05 MST"))
		fmt.Printf("Duration: %v\n", duration)
		fmt.Println("========================================")
		status = "DRY RUN"
		logVerbose(config.VerboseMode, "WhatIf mode enabled - calendar invite preview displayed, API call skipped")
	} else {
		// Normal execution - actually create the event
		logVerbose(config.VerboseMode, "Calling Graph API: POST /users/%s/events", mailbox)
		logVerbose(config.VerboseMode, "Calendar invite - Subject: %s, Start: %s, End: %s", subject, startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))
		createdEvent, err := client.Users().ByUserId(mailbox).Events().Post(ctx, event, nil)

		if err != nil {
			// Enrich error with rate limit and service error details
			enrichedErr := enrichGraphAPIError(err, logger, "createInvite")
			log.Printf("Error creating invite: %v", enrichedErr)
			status = fmt.Sprintf("%s: %v", StatusError, enrichedErr)
		} else {
			if createdEvent.GetId() != nil {
				eventID = *createdEvent.GetId()
			}
			logVerbose(config.VerboseMode, "Calendar event created successfully via Graph API")
			logVerbose(config.VerboseMode, "Event ID: %s", eventID)
			fmt.Printf("Calendar invitation created in mailbox: %s\n", mailbox)
			fmt.Printf("Subject: %s\n", subject)
			fmt.Printf("Start: %s\n", startTime.Format("2006-01-02 15:04:05 MST"))
			fmt.Printf("End: %s\n", endTime.Format("2006-01-02 15:04:05 MST"))
			fmt.Printf("Event ID: %s\n", eventID)
		}
	}

	// Write to CSV
	if logger != nil {
		logger.WriteRow([]string{ActionSendInvite, status, mailbox, subject, startTime.Format(time.RFC3339), endTime.Format(time.RFC3339), eventID})
	}
}

// checkAvailability checks a recipient's calendar availability for the next working day at 12:00-13:00 UTC.
// Used to verify calendar access and test availability queries.
func checkAvailability(ctx context.Context, client *msgraphsdk.GraphServiceClient, mailbox string, recipient string, config *Config, logger *CSVLogger) error {
	// Calculate next working day
	now := time.Now().UTC()
	nextWorkingDay := addWorkingDays(now, 1)

	// Set time to 12:00 UTC (noon)
	checkDateTime := time.Date(
		nextWorkingDay.Year(),
		nextWorkingDay.Month(),
		nextWorkingDay.Day(),
		12, 0, 0, 0,
		time.UTC,
	)

	// End time is 1 hour later (13:00 UTC)
	endDateTime := checkDateTime.Add(1 * time.Hour)

	logVerbose(config.VerboseMode, "Checking availability for %s on %s (12:00-13:00 UTC)", recipient, checkDateTime.Format("2006-01-02"))

	// Create DateTimeTimeZone objects for Graph API
	startTimeZone := models.NewDateTimeTimeZone()
	startTimeZone.SetDateTime(pointerTo(checkDateTime.Format(time.RFC3339)))
	startTimeZone.SetTimeZone(pointerTo("UTC"))

	endTimeZone := models.NewDateTimeTimeZone()
	endTimeZone.SetDateTime(pointerTo(endDateTime.Format(time.RFC3339)))
	endTimeZone.SetTimeZone(pointerTo("UTC"))

	// Create request body
	requestBody := users.NewItemCalendarGetSchedulePostRequestBody()
	requestBody.SetSchedules([]string{recipient})
	requestBody.SetStartTime(startTimeZone)
	requestBody.SetEndTime(endTimeZone)
	interval := int32(60) // 60-minute intervals
	requestBody.SetAvailabilityViewInterval(&interval)

	logVerbose(config.VerboseMode, "Calling Graph API: POST /users/%s/calendar/getSchedule", mailbox)

	// Execute API call with retry logic
	var scheduleInfo []models.ScheduleInformationable
	err := retryWithBackoff(ctx, config.MaxRetries, config.RetryDelay, func() error {
		response, apiErr := client.Users().ByUserId(mailbox).Calendar().GetSchedule().Post(ctx, requestBody, nil)
		if apiErr == nil && response != nil {
			scheduleInfo = response.GetValue()
		}
		return apiErr
	})

	if err != nil {
		// Enrich error with rate limit and service error details
		enrichedErr := enrichGraphAPIError(err, logger, "checkAvailability")
		csvRow := []string{ActionGetSchedule, fmt.Sprintf("Error: %v", enrichedErr), mailbox, recipient, checkDateTime.Format(time.RFC3339), "N/A"}
		if logger != nil {
			logger.WriteRow(csvRow)
		}
		return fmt.Errorf("error checking availability for %s: %w", recipient, enrichedErr)
	}

	logVerbose(config.VerboseMode, "API response received: %d schedule(s)", len(scheduleInfo))

	// Parse availability view
	if len(scheduleInfo) == 0 {
		errMsg := "no schedule information returned"
		csvRow := []string{ActionGetSchedule, fmt.Sprintf("Error: %s", errMsg), mailbox, recipient, checkDateTime.Format(time.RFC3339), "N/A"}
		if logger != nil {
			logger.WriteRow(csvRow)
		}
		return fmt.Errorf("no schedule information returned")
	}

	// Get availability view from first schedule
	info := scheduleInfo[0]
	availabilityView := ""
	if info.GetAvailabilityView() != nil {
		availabilityView = *info.GetAvailabilityView()
	}

	if availabilityView == "" {
		errMsg := "empty availability view returned"
		csvRow := []string{ActionGetSchedule, fmt.Sprintf("Error: %s", errMsg), mailbox, recipient, checkDateTime.Format(time.RFC3339), "N/A"}
		if logger != nil {
			logger.WriteRow(csvRow)
		}
		return fmt.Errorf("empty availability view returned")
	}

	// Interpret availability
	status := interpretAvailability(availabilityView)

	if config.OutputFormat == "json" {
		printJSON(formatScheduleOutput(scheduleInfo))
	} else {
		// Display results
		fmt.Printf("Availability Check Results:\n")
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		fmt.Printf("Organizer:     %s\n", mailbox)
		fmt.Printf("Recipient:     %s\n", recipient)
		fmt.Printf("Check Date:    %s\n", checkDateTime.Format("2006-01-02"))
		fmt.Printf("Check Time:    12:00-13:00 UTC\n")
		fmt.Printf("Status:        %s\n", status)
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")
	}

	logVerbose(config.VerboseMode, "Availability view: %s → %s", availabilityView, status)

	// Log to CSV
	if logger != nil {
		csvRow := []string{ActionGetSchedule, StatusSuccess, mailbox, recipient, checkDateTime.Format(time.RFC3339), availabilityView}
		logger.WriteRow(csvRow)
	}

	return nil
}

// interpretAvailability converts Microsoft Graph availability view codes to human-readable status.
// Availability codes: 0=Free, 1=Tentative, 2=Busy, 3=Out of Office, 4=Working Elsewhere
func interpretAvailability(view string) string {
	if len(view) == 0 {
		return "Unknown (empty response)"
	}

	// Get the first character (representing the time slot status)
	code := string(view[0])

	switch code {
	case "0":
		return "Free"
	case "1":
		return "Tentative"
	case "2":
		return "Busy"
	case "3":
		return "Out of Office"
	case "4":
		return "Working Elsewhere"
	default:
		return fmt.Sprintf("Unknown (%s)", code)
	}
}

// parseFlexibleTime parses time strings in multiple formats.
// Accepts RFC3339 (with timezone) and PowerShell sortable format (without timezone, assumes UTC).
func parseFlexibleTime(timeStr string) (time.Time, error) {
	if timeStr == "" {
		return time.Time{}, fmt.Errorf("time string is empty")
	}

	// Try RFC3339 first (with timezone)
	t, err := time.Parse(time.RFC3339, timeStr)
	if err == nil {
		return t, nil
	}

	// Try PowerShell sortable format (without timezone) - assume UTC
	t, err = time.Parse("2006-01-02T15:04:05", timeStr)
	if err == nil {
		return t.UTC(), nil
	}

	return time.Time{}, fmt.Errorf("invalid time format (expected RFC3339 like '2026-01-15T14:00:00Z' or PowerShell sortable like '2026-01-15T14:00:05')")
}

// addWorkingDays adds a specified number of working days (Monday-Friday) to the given time.
// Skips weekends (Saturday and Sunday).
func addWorkingDays(t time.Time, days int) time.Time {
	if days <= 0 {
		return t
	}

	result := t
	daysAdded := 0

	for daysAdded < days {
		result = result.Add(24 * time.Hour)

		// Check if this is a working day (Monday=1, Friday=5)
		weekday := result.Weekday()
		if weekday != time.Saturday && weekday != time.Sunday {
			daysAdded++
		}
	}

	return result
}
