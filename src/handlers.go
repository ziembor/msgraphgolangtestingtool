package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/microsoftgraph/msgraph-sdk-go/users"
)

// executeAction dispatches to the appropriate action handler based on config.Action.
// Supported actions are: getevents, sendmail, sendinvite, and getinbox.
//
// For sendmail action, if no recipients are specified, the email is sent to the
// mailbox owner (self). All actions log their operations to the provided CSV logger.
//
// Returns an error if the action fails or if the action name is unknown.
func executeAction(ctx context.Context, client *msgraphsdk.GraphServiceClient, config *Config, logger *CSVLogger) error {
	switch config.Action {
	case ActionGetEvents:
		if err := listEvents(ctx, client, config.Mailbox, config.Count, config, logger); err != nil {
			return fmt.Errorf("failed to list events: %w", err)
		}
	case ActionSendMail:
		// If no recipients specified at all, default 'to' to the sender mailbox
		if len(config.To) == 0 && len(config.Cc) == 0 && len(config.Bcc) == 0 {
			config.To = []string{config.Mailbox}
		}

		sendEmail(ctx, client, config.Mailbox, config.To, config.Cc, config.Bcc, config.Subject, config.Body, config.BodyHTML, config.AttachmentFiles, config, logger)
	case ActionSendInvite:
		// Use Subject for calendar invite
		// For backward compatibility, if InviteSubject is set, use it instead
		inviteSubject := config.Subject
		if config.InviteSubject != "" {
			inviteSubject = config.InviteSubject
		}
		// If using default email subject, change to default calendar invite subject
		if inviteSubject == "Automated Tool Notification" {
			inviteSubject = "It's testing event"
		}
		createInvite(ctx, client, config.Mailbox, inviteSubject, config.StartTime, config.EndTime, config, logger)
	case ActionGetInbox:
		if err := listInbox(ctx, client, config.Mailbox, config.Count, config, logger); err != nil {
			return fmt.Errorf("failed to list inbox: %w", err)
		}
	case ActionGetSchedule:
		if err := checkAvailability(ctx, client, config.Mailbox, config.To[0], config, logger); err != nil {
			return fmt.Errorf("failed to check availability: %w", err)
		}
	case ActionExportInbox:
		if err := exportInbox(ctx, client, config.Mailbox, config.Count, config, logger); err != nil {
			return fmt.Errorf("failed to export inbox: %w", err)
		}
	case ActionSearchAndExport:
		if err := searchAndExport(ctx, client, config.Mailbox, config.MessageID, config, logger); err != nil {
			return fmt.Errorf("failed to search and export: %w", err)
		}
	default:
		return fmt.Errorf("unknown action: %s", config.Action)
	}

	return nil
}

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

// exportInbox exports messages from the inbox to JSON files
func searchAndExport(ctx context.Context, client *msgraphsdk.GraphServiceClient, mailbox string, messageID string, config *Config, logger *CSVLogger) error {
	// Configure request with filter
	// Note: We search the whole mailbox (Messages endpoint), not just Inbox
	// SECURITY: Escape single quotes for OData filter (defense-in-depth)
	// Even though validateMessageID() blocks quotes, we escape as an additional safeguard
	escapedMessageID := strings.ReplaceAll(messageID, "'", "''")
	filter := fmt.Sprintf("internetMessageId eq '%s'", escapedMessageID)
	requestConfig := &users.ItemMessagesRequestBuilderGetRequestConfiguration{
		QueryParameters: &users.ItemMessagesRequestBuilderGetQueryParameters{
			Filter: &filter,
			Select: []string{"id", "internetMessageId", "subject", "receivedDateTime", "from", "toRecipients", "ccRecipients", "bccRecipients", "body", "hasAttachments"},
		},
	}

	logVerbose(config.VerboseMode, "Calling Graph API: GET /users/%s/messages?$filter=%s", mailbox, filter)

	// Execute API call with retry logic
	var getValueFunc func() []models.Messageable
	err := retryWithBackoff(ctx, config.MaxRetries, config.RetryDelay, func() error {
		apiResult, apiErr := client.Users().ByUserId(mailbox).Messages().Get(ctx, requestConfig)
		if apiErr == nil {
			getValueFunc = apiResult.GetValue
		}
		return apiErr
	})

	if err != nil {
		enrichedErr := enrichGraphAPIError(err, logger, "searchAndExport")
		return fmt.Errorf("error searching message for %s: %w", mailbox, enrichedErr)
	}

	messages := getValueFunc()
	messageCount := len(messages)

	logVerbose(config.VerboseMode, "API response received: %d messages", messageCount)

	if messageCount == 0 {
		if config.OutputFormat == "json" {
			printJSON([]interface{}{}) // Empty array
		} else {
			fmt.Printf("No message found with Internet Message ID: %s\n", messageID)
		}
		if logger != nil {
			logger.WriteRow([]string{ActionSearchAndExport, StatusSuccess, mailbox, "Message not found", messageID})
		}
		return nil
	}

	// Print JSON output if requested
	if config.OutputFormat == "json" {
		printJSON(formatMessagesOutput(messages))
	}

	// Create export directory
	exportDir, err := createExportDir()
	if err != nil {
		return err
	}
	
	if config.OutputFormat != "json" {
		fmt.Printf("Export directory: %s\n", exportDir)
	}

	// Export found messages (usually 1, but duplicates technically possible in some scenarios)
	for _, message := range messages {
		if err := exportMessageToJSON(message, exportDir, config); err != nil {
			return fmt.Errorf("failed to export message: %w", err)
		}
		if config.OutputFormat != "json" {
			fmt.Printf("Successfully exported message: %s\n", *message.GetSubject())
		}
		if logger != nil {
			logger.WriteRow([]string{ActionSearchAndExport, StatusSuccess, mailbox, "Exported successfully", *message.GetId()})
		}
	}

	return nil
}

// exportMessageToJSON serializes a message to JSON and saves it to a file
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

// createFileAttachments reads files and creates Graph API attachment objects
// parseFlexibleTime parses a time string accepting multiple formats
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

	return time.Time{}, fmt.Errorf("invalid time format (expected RFC3339 like '2026-01-15T14:00:00Z' or PowerShell sortable like '2026-01-15T14:00:00')")
}

// addWorkingDays adds a specified number of working days (Monday-Friday) to the given time.
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

// Status constants
const (
	StatusSuccess = "Success"
	StatusError   = "Error"
)

// Output helper functions

// printJSON marshals the data to JSON and prints it to stdout
func printJSON(data interface{}) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON output: %v\n", err)
	}
}

// formatEventsOutput converts a list of Eventable items to a JSON-friendly slice of maps
func formatEventsOutput(events []models.Eventable) []map[string]interface{} {
	var output []map[string]interface{}
	for _, event := range events {
		eventMap := make(map[string]interface{})
		if event.GetId() != nil {
			eventMap["id"] = *event.GetId()
		}
		if event.GetSubject() != nil {
			eventMap["subject"] = *event.GetSubject()
		}
		if event.GetStart() != nil && event.GetStart().GetDateTime() != nil {
			eventMap["start"] = *event.GetStart().GetDateTime()
		}
		if event.GetEnd() != nil && event.GetEnd().GetDateTime() != nil {
			eventMap["end"] = *event.GetEnd().GetDateTime()
		}
		if event.GetOrganizer() != nil && event.GetOrganizer().GetEmailAddress() != nil {
			eventMap["organizer"] = extractEmailAddress(event.GetOrganizer().GetEmailAddress())
		}
		output = append(output, eventMap)
	}
	return output
}

// formatMessagesOutput converts a list of Messageable items to a JSON-friendly slice of maps
func formatMessagesOutput(messages []models.Messageable) []map[string]interface{} {
	var output []map[string]interface{}
	for _, message := range messages {
		msgMap := make(map[string]interface{})
		if message.GetId() != nil {
			msgMap["id"] = *message.GetId()
		}
		if message.GetSubject() != nil {
			msgMap["subject"] = *message.GetSubject()
		}
		if message.GetReceivedDateTime() != nil {
			msgMap["receivedDateTime"] = message.GetReceivedDateTime().Format(time.RFC3339)
		}
		if message.GetFrom() != nil && message.GetFrom().GetEmailAddress() != nil {
			msgMap["from"] = extractEmailAddress(message.GetFrom().GetEmailAddress())
		}
		if message.GetToRecipients() != nil {
			msgMap["toRecipients"] = extractRecipients(message.GetToRecipients())
		}
		output = append(output, msgMap)
	}
	return output
}

// formatScheduleOutput converts a list of ScheduleInformationable items to a JSON-friendly structure
func formatScheduleOutput(schedules []models.ScheduleInformationable) []map[string]interface{} {
	var output []map[string]interface{}
	for _, schedule := range schedules {
		schMap := make(map[string]interface{})
		if schedule.GetScheduleId() != nil {
			schMap["scheduleId"] = *schedule.GetScheduleId()
		}
		if schedule.GetAvailabilityView() != nil {
			schMap["availabilityView"] = *schedule.GetAvailabilityView()
			schMap["availabilityStatus"] = interpretAvailability(*schedule.GetAvailabilityView())
		}
		// Include working hours if available
		if schedule.GetWorkingHours() != nil {
			wh := schedule.GetWorkingHours()
			whMap := make(map[string]interface{})
			if wh.GetStartTime() != nil {
				whMap["startTime"] = *wh.GetStartTime()
			}
			if wh.GetEndTime() != nil {
				whMap["endTime"] = *wh.GetEndTime()
			}
			schMap["workingHours"] = whMap
		}
		output = append(output, schMap)
	}
	return output
}
