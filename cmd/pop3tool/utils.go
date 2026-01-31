package main

// maskUsername masks a username for safe logging.
// Shows first 2 and last 2 characters with **** in between.
func maskUsername(username string) string {
	if len(username) <= 4 {
		return "****"
	}
	return username[:2] + "****" + username[len(username)-2:]
}
