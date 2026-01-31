package main

// maskUsername masks a username for safe logging.
// Shows first 2 and last 2 characters with **** in between.
func maskUsername(username string) string {
	if len(username) <= 4 {
		return "****"
	}
	return username[:2] + "****" + username[len(username)-2:]
}

// maskPassword masks a password for safe logging.
// Shows first 2 and last 2 characters with **** in between.
func maskPassword(password string) string {
	if len(password) == 0 {
		return ""
	}
	if len(password) <= 4 {
		return "****"
	}
	return password[:2] + "****" + password[len(password)-2:]
}

// maskAccessToken masks an access token for safe logging.
// Shows first 8 and last 4 characters with ... in between.
func maskAccessToken(token string) string {
	if len(token) == 0 {
		return ""
	}
	if len(token) <= 16 {
		return token[:len(token)/2] + "..." + token[len(token)/2:]
	}
	return token[:8] + "..." + token[len(token)-4:]
}
