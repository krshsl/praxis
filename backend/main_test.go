package main

import (
	"net/http/httptest"
	"testing"

	"github.com/spf13/viper"
	svc "github.com/krshsl/praxis/backend/services"
)

func TestCheckOrigin(t *testing.T) {
	tests := []struct {
		name           string
		allowedOrigins string
		requestOrigin  string
		expected       bool
	}{
		{
			name:           "Allowed origin - exact match",
			allowedOrigins: "http://localhost,http://example.com",
			requestOrigin:  "http://localhost",
			expected:       true,
		},
		{
			name:           "Allowed origin - second in list",
			allowedOrigins: "http://localhost,http://example.com",
			requestOrigin:  "http://example.com",
			expected:       true,
		},
		{
			name:           "Disallowed origin",
			allowedOrigins: "http://localhost,http://example.com",
			requestOrigin:  "http://malicious.com",
			expected:       false,
		},
		{
			name:           "Empty allowed origins - deny all",
			allowedOrigins: "",
			requestOrigin:  "http://localhost",
			expected:       false,
		},
		{
			name:           "Origin with whitespace in config",
			allowedOrigins: "http://localhost, http://example.com",
			requestOrigin:  "http://example.com",
			expected:       true,
		},
		{
			name:           "Port-specific origin allowed",
			allowedOrigins: "http://localhost:5173",
			requestOrigin:  "http://localhost:5173",
			expected:       true,
		},
		{
			name:           "Port mismatch - deny",
			allowedOrigins: "http://localhost:5173",
			requestOrigin:  "http://localhost:8080",
			expected:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset viper config for each test
			viper.Reset()
			viper.Set("websocket.allowed_origins", tt.allowedOrigins)

			// Create a request with the test origin
			req := httptest.NewRequest("GET", "/api/v1/ws", nil)
			req.Header.Set("Origin", tt.requestOrigin)

			// Test the CheckOrigin function with allowed origins from config
			allowed := viper.GetString("websocket.allowed_origins")
			result := svc.CheckOrigin(req, allowed)

			if result != tt.expected {
				t.Errorf("checkOrigin() = %v, expected %v for origin %s with allowed origins %s",
					result, tt.expected, tt.requestOrigin, tt.allowedOrigins)
			}
		})
	}
}
