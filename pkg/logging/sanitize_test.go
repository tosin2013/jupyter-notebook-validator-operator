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

package logging

import (
	"errors"
	"strings"
	"testing"
)

func TestSanitizeURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "HTTPS URL with credentials",
			input:    "https://user:password@github.com/org/repo.git",
			expected: "https://github.com/org/repo.git",
		},
		{
			name:     "HTTPS URL with token",
			input:    "https://token@github.com/org/repo.git",
			expected: "https://github.com/org/repo.git",
		},
		{
			name:     "HTTPS URL without credentials",
			input:    "https://github.com/org/repo.git",
			expected: "https://github.com/org/repo.git",
		},
		{
			name:     "SSH URL (not a valid HTTP URL, returns as-is after parsing)",
			input:    "git@github.com:org/repo.git",
			expected: "[invalid-url]", // SSH URLs are not valid HTTP URLs
		},
		{
			name:     "Empty URL",
			input:    "",
			expected: "",
		},
		{
			name:     "Invalid URL (gets URL-encoded by parser)",
			input:    "not a valid url",
			expected: "not%20a%20valid%20url", // url.Parse encodes spaces
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeURL(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeURL(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeError(t *testing.T) {
	tests := []struct {
		name               string
		err                error
		sensitiveStrings   []string
		expectedContains   string
		expectedNotContain string
	}{
		{
			name:               "Error with password",
			err:                errors.New("failed to connect with password: secret123"),
			sensitiveStrings:   []string{"secret123"},
			expectedContains:   "[REDACTED]",
			expectedNotContain: "secret123",
		},
		{
			name:               "Error with multiple sensitive strings",
			err:                errors.New("auth failed: user=admin token=abc123"),
			sensitiveStrings:   []string{"admin", "abc123"},
			expectedContains:   "[REDACTED]",
			expectedNotContain: "admin",
		},
		{
			name:               "Nil error",
			err:                nil,
			sensitiveStrings:   []string{"secret"},
			expectedContains:   "",
			expectedNotContain: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeError(tt.err, tt.sensitiveStrings...)
			if tt.err == nil {
				if result != nil {
					t.Errorf("SanitizeError(nil) should return nil, got %v", result)
				}
				return
			}

			resultStr := result.Error()
			if tt.expectedContains != "" && !strings.Contains(resultStr, tt.expectedContains) {
				t.Errorf("SanitizeError() result should contain %q, got %q", tt.expectedContains, resultStr)
			}
			if tt.expectedNotContain != "" && strings.Contains(resultStr, tt.expectedNotContain) {
				t.Errorf("SanitizeError() result should not contain %q, got %q", tt.expectedNotContain, resultStr)
			}
		})
	}
}

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Long string",
			input:    "verylongsecrettoken123456",
			expected: "ve***56",
		},
		{
			name:     "Short string",
			input:    "abc",
			expected: "***",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Exactly 4 characters",
			input:    "abcd",
			expected: "***",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeString(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeString(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeSecretData(t *testing.T) {
	data := map[string][]byte{
		"username": []byte("admin"),
		"password": []byte("secret123"),
		"token":    []byte("abc123xyz"),
	}

	result := SanitizeSecretData(data)

	// All values should be redacted
	for key, value := range result {
		if value != "[REDACTED]" {
			t.Errorf("SanitizeSecretData() key %q should be [REDACTED], got %q", key, value)
		}
	}

	// All keys should be preserved
	if len(result) != len(data) {
		t.Errorf("SanitizeSecretData() should preserve all keys, got %d keys, want %d", len(result), len(data))
	}
}

func TestSanitizeEnvVars(t *testing.T) {
	envVars := map[string]string{
		"HOME":                  "/home/user",
		"PATH":                  "/usr/bin:/bin",
		"AWS_ACCESS_KEY_ID":     "AKIAIOSFODNN7EXAMPLE",
		"AWS_SECRET_ACCESS_KEY": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		"DB_PASSWORD":           "secret123",
		"API_KEY":               "abc123xyz",
		"NORMAL_VAR":            "normal_value",
	}

	result := SanitizeEnvVars(envVars)

	// Check that sensitive vars are redacted
	sensitiveKeys := []string{"AWS_SECRET_ACCESS_KEY", "DB_PASSWORD", "API_KEY"}
	for _, key := range sensitiveKeys {
		if result[key] != "[REDACTED]" {
			t.Errorf("SanitizeEnvVars() key %q should be [REDACTED], got %q", key, result[key])
		}
	}

	// Check that non-sensitive vars are preserved
	nonSensitiveKeys := []string{"HOME", "PATH", "NORMAL_VAR"}
	for _, key := range nonSensitiveKeys {
		if result[key] != envVars[key] {
			t.Errorf("SanitizeEnvVars() key %q should be preserved, got %q, want %q", key, result[key], envVars[key])
		}
	}
}

func TestSanitizeCommand(t *testing.T) {
	tests := []struct {
		name               string
		input              string
		expectedNotContain []string
		expectedContain    []string
	}{
		{
			name:               "Git clone with credentials",
			input:              "git clone https://user:password@github.com/org/repo.git",
			expectedNotContain: []string{"user", "password"},
			expectedContain:    []string{"[REDACTED]"},
		},
		{
			name:               "SSH private key",
			input:              "echo '-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA...\n-----END RSA PRIVATE KEY-----' > key",
			expectedNotContain: []string{"MIIEpAIBAAKCAQEA"},
			expectedContain:    []string{"[REDACTED-SSH-KEY]"},
		},
		{
			name:               "Password flag",
			input:              "mysql --password=secret123 -h localhost",
			expectedNotContain: []string{"secret123"},
			expectedContain:    []string{"[REDACTED]"},
		},
		{
			name:               "Environment variable with password",
			input:              "export DB_PASSWORD=secret123",
			expectedNotContain: []string{"secret123"},
			expectedContain:    []string{"[REDACTED]"},
		},
		{
			name:               "Normal command",
			input:              "ls -la /home/user",
			expectedNotContain: []string{},
			expectedContain:    []string{"ls -la /home/user"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeCommand(tt.input)

			for _, notContain := range tt.expectedNotContain {
				if strings.Contains(result, notContain) {
					t.Errorf("SanitizeCommand() result should not contain %q, got %q", notContain, result)
				}
			}

			for _, contain := range tt.expectedContain {
				if !strings.Contains(result, contain) {
					t.Errorf("SanitizeCommand() result should contain %q, got %q", contain, result)
				}
			}
		})
	}
}

func TestLogFields_Sanitize(t *testing.T) {
	fields := LogFields{
		"namespace": "default",
		"name":      "test-job",
		"password":  "secret123",
		"token":     "abc123xyz",
		"normalKey": "normalValue",
	}

	result := fields.Sanitize("password", "token")

	// Check sensitive keys are redacted
	if result["password"] != "[REDACTED]" {
		t.Errorf("LogFields.Sanitize() password should be [REDACTED], got %v", result["password"])
	}
	if result["token"] != "[REDACTED]" {
		t.Errorf("LogFields.Sanitize() token should be [REDACTED], got %v", result["token"])
	}

	// Check non-sensitive keys are preserved
	if result["namespace"] != "default" {
		t.Errorf("LogFields.Sanitize() namespace should be preserved, got %v", result["namespace"])
	}
	if result["normalKey"] != "normalValue" {
		t.Errorf("LogFields.Sanitize() normalKey should be preserved, got %v", result["normalKey"])
	}
}

func TestLogFields_ToKeyValues(t *testing.T) {
	fields := LogFields{
		"key1": "value1",
		"key2": 42,
		"key3": true,
	}

	result := fields.ToKeyValues()

	// Should have 6 elements (3 key-value pairs)
	if len(result) != 6 {
		t.Errorf("LogFields.ToKeyValues() should return 6 elements, got %d", len(result))
	}

	// Check that keys and values are present
	hasKey1 := false
	hasValue1 := false
	for i := 0; i < len(result); i += 2 {
		if result[i] == "key1" {
			hasKey1 = true
			if result[i+1] == "value1" {
				hasValue1 = true
			}
		}
	}

	if !hasKey1 || !hasValue1 {
		t.Errorf("LogFields.ToKeyValues() should contain key1=value1")
	}
}
