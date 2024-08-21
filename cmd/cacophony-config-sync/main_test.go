package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMapServerSettingsToConfig(t *testing.T) {
	syncService := &SyncService{}

	testCases := []struct {
		name           string
		serverSettings map[string]interface{}
		expected       map[string]interface{}
	}{
		{
			name: "Map Windows settings",
			serverSettings: map[string]interface{}{
				"windows": map[string]interface{}{
					"startRecording": "+30m",
					"stopRecording":  "-30m",
					"powerOn":        "12:00",
					"powerOff":       "13:00",
					"updated":        "2023-08-22T10:00:00Z",
				},
			},
			expected: map[string]interface{}{
				"windows": map[string]interface{}{
					"start-recording": "+30m",
					"stop-recording":  "-30m",
					"power-on":        "12:00",
					"power-off":       "13:00",
					"updated":         time.Date(2023, 8, 22, 10, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			name: "Map Location settings",
			serverSettings: map[string]interface{}{
				"location": map[string]interface{}{
					"lat": "12.345",
					"lng": "67.89",
				},
			},
			expected: map[string]interface{}{
				"location": map[string]interface{}{
					"latitude":  "12.345",
					"longitude": "67.89",
				},
			},
		},
		{
			name:           "Empty server settings",
			serverSettings: map[string]interface{}{},
			expected:       map[string]interface{}{},
		},
		{
			name: "Unknown section",
			serverSettings: map[string]interface{}{
				"unknown": map[string]interface{}{
					"key": "value",
				},
			},
			expected: map[string]interface{}{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := syncService.mapServerSettingsToConfig(tc.serverSettings)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestFilterUnchangedSettings(t *testing.T) {
	syncService := &SyncService{}

	now := time.Now()
	testCases := []struct {
		name           string
		mappedSettings map[string]interface{}
		deviceSettings map[string]interface{}
		expected       map[string]interface{}
	}{
		{
			name: "No changes",
			mappedSettings: map[string]interface{}{
				"windows": map[string]interface{}{
					"start-recording": "+30m",
					"updated":         now,
				},
			},
			deviceSettings: map[string]interface{}{
				"windows": map[string]interface{}{
					"startRecording": "+30m",
					"updated":        now,
				},
			},
			expected: map[string]interface{}{},
		},
		{
			name: "Updated time is older",
			mappedSettings: map[string]interface{}{
				"windows": map[string]interface{}{
					"start-recording": "+30m",
					"updated":         time.Now().Add(-1 * time.Hour),
				},
			},
			deviceSettings: map[string]interface{}{
				"windows": map[string]interface{}{
					"startRecording": "+30m",
					"updated":        now,
				},
			},
			expected: map[string]interface{}{},
		},
		{
			name: "Updated time is newer",
			mappedSettings: map[string]interface{}{
				"windows": map[string]interface{}{
					"start-recording": "+40m",
					"updated":         now,
				},
			},
			deviceSettings: map[string]interface{}{
				"windows": map[string]interface{}{
					"startRecording": "+30m",
					"updated":        time.Now().Add(-1 * time.Hour),
				},
			},
			expected: map[string]interface{}{
				"windows": map[string]interface{}{
					"start-recording": "+40m",
					"updated":         now,
				},
			},
		},
		{
			name: "New section",
			mappedSettings: map[string]interface{}{
				"windows": map[string]interface{}{
					"start-recording": "+30m",
					"updated":         now,
				},
				"location": map[string]interface{}{
					"latitude":  12.345,
					"longitude": 67.890,
				},
			},
			deviceSettings: map[string]interface{}{
				"windows": map[string]interface{}{
					"startRecording": "+30m",
					"updated":        now,
				},
			},
			expected: map[string]interface{}{
				"location": map[string]interface{}{
					"latitude":  12.345,
					"longitude": 67.890,
				},
			},
		},
		{
			name: "Ignore unknown sections",
			mappedSettings: map[string]interface{}{
				"windows": map[string]interface{}{
					"start-recording": "+30m",
					"updated":         time.Now(),
				},
				"unknown": map[string]interface{}{
					"key":     "value",
					"updated": time.Now(),
				},
			},
			deviceSettings: map[string]interface{}{
				"windows": map[string]interface{}{
					"startRecording": "+30m",
					"updated":        time.Now(),
				},
			},
			expected: map[string]interface{}{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := syncService.filterUnchangedSettings(tc.mappedSettings, tc.deviceSettings)
			assert.Equal(t, tc.expected, result)
		})
	}
}
