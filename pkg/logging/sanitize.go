/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package logging provides utilities for structured logging and log sanitization
// Based on ADR-010: Observability and Monitoring Strategy
package logging

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// SanitizeURL removes credentials from URLs for logging
// This prevents leaking usernames, passwords, and tokens in logs
func SanitizeURL(rawURL string) string {
	if rawURL == "" {
		return ""
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return "[invalid-url]"
	}

	// Remove user info (credentials)
	u.User = nil

	return u.String()
}

// SanitizeError removes sensitive information from error messages
// This prevents leaking credentials, tokens, and other sensitive data in error logs
func SanitizeError(err error, sensitiveStrings ...string) error {
	if err == nil {
		return nil
	}

	msg := err.Error()
	for _, sensitive := range sensitiveStrings {
		if sensitive != "" && len(sensitive) > 0 {
			msg = strings.ReplaceAll(msg, sensitive, "[REDACTED]")
		}
	}

	return fmt.Errorf("%s", msg)
}

// SanitizeString removes or masks sensitive information from a string
// Shows first 2 and last 2 characters for debugging while hiding the middle
func SanitizeString(value string) string {
	if value == "" {
		return ""
	}

	if len(value) <= 4 {
		return "***"
	}

	// Show first 2 and last 2 characters
	return value[:2] + "***" + value[len(value)-2:]
}

// SanitizeSecretData sanitizes secret data for logging
// Returns a map with keys preserved but values masked
func SanitizeSecretData(data map[string][]byte) map[string]string {
	sanitized := make(map[string]string)
	for key := range data {
		sanitized[key] = "[REDACTED]"
	}
	return sanitized
}

// SanitizeEnvVars sanitizes environment variables for logging
// Masks values of known sensitive environment variable names
func SanitizeEnvVars(envVars map[string]string) map[string]string {
	sensitiveKeys := []string{
		"password", "token", "secret", "key", "credential",
		"api_key", "apikey", "auth", "bearer",
		"AWS_SECRET_ACCESS_KEY", "AWS_SESSION_TOKEN",
		"GCP_SERVICE_ACCOUNT_KEY", "AZURE_CLIENT_SECRET",
		"DB_PASSWORD", "DATABASE_PASSWORD",
	}

	sanitized := make(map[string]string)
	for key, value := range envVars {
		keyLower := strings.ToLower(key)
		isSensitive := false

		for _, sensitiveKey := range sensitiveKeys {
			if strings.Contains(keyLower, strings.ToLower(sensitiveKey)) {
				isSensitive = true
				break
			}
		}

		if isSensitive {
			sanitized[key] = "[REDACTED]"
		} else {
			sanitized[key] = value
		}
	}

	return sanitized
}

// SanitizeCommand sanitizes shell commands for logging
// Removes sensitive data from command strings
func SanitizeCommand(command string) string {
	if command == "" {
		return ""
	}

	// Patterns to match and redact
	patterns := []struct {
		regex       *regexp.Regexp
		replacement string
	}{
		// Git URLs with credentials: https://user:pass@github.com/repo.git
		{
			regex:       regexp.MustCompile(`(https?://)[^:@]+:[^@]+@`),
			replacement: "$1[REDACTED]:[REDACTED]@",
		},
		// SSH private keys
		{
			regex:       regexp.MustCompile(`-----BEGIN [A-Z ]+PRIVATE KEY-----[\s\S]*?-----END [A-Z ]+PRIVATE KEY-----`),
			replacement: "[REDACTED-SSH-KEY]",
		},
		// Base64 encoded data (likely credentials)
		{
			regex:       regexp.MustCompile(`([A-Za-z0-9+/]{40,}={0,2})`),
			replacement: "[REDACTED-BASE64]",
		},
		// Password flags: --password=secret, -p secret
		{
			regex:       regexp.MustCompile(`(--password[= ]|--token[= ]|-p )[^\s]+`),
			replacement: "$1[REDACTED]",
		},
		// Environment variable assignments: PASSWORD=secret
		{
			regex:       regexp.MustCompile(`([A-Z_]*(?:PASSWORD|TOKEN|SECRET|KEY)[A-Z_]*=)[^\s]+`),
			replacement: "$1[REDACTED]",
		},
	}

	sanitized := command
	for _, pattern := range patterns {
		sanitized = pattern.regex.ReplaceAllString(sanitized, pattern.replacement)
	}

	return sanitized
}

// SanitizeLogMessage sanitizes a log message by removing sensitive data
// This is a catch-all function that applies multiple sanitization rules
func SanitizeLogMessage(message string, sensitiveStrings ...string) string {
	if message == "" {
		return ""
	}

	sanitized := message

	// Apply custom sensitive strings
	for _, sensitive := range sensitiveStrings {
		if sensitive != "" && len(sensitive) > 0 {
			sanitized = strings.ReplaceAll(sanitized, sensitive, "[REDACTED]")
		}
	}

	// Apply command sanitization (catches URLs, keys, passwords)
	sanitized = SanitizeCommand(sanitized)

	return sanitized
}

// LogFields is a helper type for structured logging fields
type LogFields map[string]interface{}

// Sanitize sanitizes all string values in the log fields
func (f LogFields) Sanitize(sensitiveKeys ...string) LogFields {
	sanitized := make(LogFields)

	for key, value := range f {
		// Check if key is sensitive
		isSensitive := false
		keyLower := strings.ToLower(key)
		for _, sensitiveKey := range sensitiveKeys {
			if strings.Contains(keyLower, strings.ToLower(sensitiveKey)) {
				isSensitive = true
				break
			}
		}

		if isSensitive {
			sanitized[key] = "[REDACTED]"
		} else {
			// Sanitize string values
			if strValue, ok := value.(string); ok {
				sanitized[key] = SanitizeLogMessage(strValue)
			} else {
				sanitized[key] = value
			}
		}
	}

	return sanitized
}

// ToKeyValues converts LogFields to a flat slice of key-value pairs for logr
func (f LogFields) ToKeyValues() []interface{} {
	kv := make([]interface{}, 0, len(f)*2)
	for key, value := range f {
		kv = append(kv, key, value)
	}
	return kv
}
