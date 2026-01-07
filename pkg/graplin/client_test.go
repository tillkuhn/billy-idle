package graplin

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMeasurement_String(t *testing.T) {
	tests := []struct {
		name        string
		measurement Measurement
		contains    []string
	}{
		{
			name: "measurement with tags, fields, and timestamp",
			measurement: Measurement{
				Measurement: "weather",
				Tags: map[string]string{
					"location": "us-midwest",
					"sensor":   "temp-m01",
				},
				Fields: map[string]interface{}{
					"temperature": 82.3,
					"humidity":    71,
					"condition":   "sunny",
					"alert":       false,
				},
				Timestamp: time.Date(2025, 1, 6, 12, 0, 0, 0, time.UTC),
			},
			contains: []string{
				"weather",
				"location=us-midwest",
				"sensor=temp-m01",
				"temperature=82.300000",
				"humidity=71i",
				"condition=\"sunny\"",
				"alert=false",
				"1736164800000000000",
			},
		},
		{
			name: "measurement with no tags",
			measurement: Measurement{
				Measurement: "cpu",
				Fields: map[string]interface{}{
					"usage": 45.5,
				},
			},
			contains: []string{
				"cpu usage=45.500000",
			},
		},
		{
			name: "measurement with tags but no timestamp",
			measurement: Measurement{
				Measurement: "memory",
				Tags: map[string]string{
					"host": "server01",
				},
				Fields: map[string]interface{}{
					"free": 1024,
					"used": 2048,
				},
			},
			contains: []string{
				"memory,host=server01",
				"free=1024i",
				"used=2048i",
			},
		},
		{
			name: "measurement with integer field types",
			measurement: Measurement{
				Measurement: "counters",
				Fields: map[string]interface{}{
					"int_field":   42,
					"int32_field": int32(32),
					"int64_field": int64(64),
					"uint_field":  uint(10),
					"uint32":      uint32(32),
					"uint64":      uint64(64),
				},
			},
			contains: []string{
				"counters",
				"int_field=42i",
				"int32_field=32i",
				"int64_field=64i",
				"uint_field=10u",
				"uint32=32u",
				"uint64=64u",
			},
		},
		{
			name: "measurement with empty tags",
			measurement: Measurement{
				Measurement: "test",
				Tags:        map[string]string{},
				Fields: map[string]interface{}{
					"value": "test",
				},
			},
			contains: []string{
				"test value=\"test\"",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.measurement.String()
			for _, expected := range tt.contains {
				if !containsString(got, expected) {
					t.Errorf("Measurement.String() = %v, want to contain %v", got, expected)
				}
			}
		})
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestNewClient(t *testing.T) {
	host := "https://example.com"
	user := "cloud9"
	token := "test_token"

	client := NewClient(WithHost(host), WithAuth(fmt.Sprintf("%s:%s", user, token)), WithDebug(true))

	if !strings.HasPrefix(client.endpoint, host) {
		t.Errorf("NewClient() endpoint = %v, should be prefixed with %v", client.endpoint, host)
	}
	if client.user != user {
		t.Errorf("NewClient() user = %v, want %v", client.user, user)
	}
	if client.token != token {
		t.Errorf("NewClient() token = %v, want %v", client.token, token)
	}
	if client.httpClient == nil {
		t.Error("NewClient() httpClient should not be nil")
	}
}

func TestClient_Push(t *testing.T) {
	measurement := Measurement{
		Measurement: "test",
		Tags: map[string]string{
			"env": "test",
		},
		Fields: map[string]interface{}{
			"value": 42.5,
		},
		Timestamp: time.Date(2025, 1, 6, 12, 0, 0, 0, time.UTC),
	}
	expectedPayload := measurement.String()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		// Verify content type
		if contentType := r.Header.Get("Content-Type"); contentType != "text/plain" {
			t.Errorf("Expected Content-Type text/plain, got %s", contentType)
		}

		// Verify basic auth
		username, password, ok := r.BasicAuth()
		if !ok || username != "cloud9" || password != "test_token" {
			t.Errorf("Expected basic auth with username=cloud9 and password=test_token, got %s:%s:%v", username, password, ok)
		}

		// Verify payload
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("Error reading request body: %v", err)
		}

		if string(body) != expectedPayload {
			t.Errorf("Expected payload %s, got %s", expectedPayload, string(body))
		}

		// Send success response
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	client := NewClient(WithHost(server.URL), WithAuth(fmt.Sprintf("%s:%s", "cloud9", "test_token")), WithDebug(true))

	err := client.Push(context.Background(), measurement)
	if err != nil {
		t.Errorf("Push() returned error: %v", err)
	}
}

func TestClient_Push_Error(t *testing.T) {
	measurement := Measurement{
		Measurement: "test",
		Fields: map[string]interface{}{
			"value": 42,
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Send error response
		w.WriteHeader(http.StatusBadRequest)
		_, err := w.Write([]byte("Bad Request"))
		if err != nil {
			t.Fail()
		}
	}))
	defer server.Close()

	client := NewClient(WithHost(server.URL), WithAuth(fmt.Sprintf("%s:%s", "cloud9", "test_token")), WithDebug(true))

	err := client.Push(context.Background(), measurement)
	if err == nil {
		t.Error("Expected Push() to return error")
	} else if !strings.Contains(err.Error(), "request failed with status code: 400") {
		t.Errorf("Expected status code error, got: %v", err)
	}
}

func TestClient_ErrorCount(t *testing.T) {
	measurement := Measurement{
		Measurement: "test",
		Fields: map[string]interface{}{
			"value": 42,
		},
	}

	client := NewClient(WithHost("http://invalid-host-that-does-not-exist.example.com"), WithAuth("user:token"))

	// Initially error count should be 0
	if count := client.ErrorCount(); count != 0 {
		t.Errorf("Expected initial error count to be 0, got %d", count)
	}

	// Make a request that will fail
	err := client.Push(context.Background(), measurement)
	if err == nil {
		t.Error("Expected Push() to return error due to invalid host")
	}

	// Error count should now be 1
	if count := client.ErrorCount(); count != 1 {
		t.Errorf("Expected error count to be 1 after failed request, got %d", count)
	}

	// Make another failing request
	err = client.Push(context.Background(), measurement)
	if err == nil {
		t.Error("Expected Push() to return error due to invalid host")
	}

	// Error count should now be 2
	if count := client.ErrorCount(); count != 2 {
		t.Errorf("Expected error count to be 2 after second failed request, got %d", count)
	}
}
