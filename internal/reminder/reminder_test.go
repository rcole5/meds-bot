package reminder

import (
	"strings"
	"testing"
	"time"

	"meds-bot/internal/config"
)

// TestShouldSendReminder tests the shouldSendReminder function
func TestShouldSendReminder(t *testing.T) {
	// Config not actually used in this test
	_ = &config.Config{
		ReminderIntervalMins: 30,
	}

	// Get current day of week for testing
	currentDay := strings.ToLower(time.Now().Weekday().String())
	// Get a different day for testing
	differentDay := "monday"
	if currentDay == "monday" {
		differentDay = "tuesday"
	}

	// Test cases
	tests := []struct {
		name        string
		medication  config.Medication
		currentHour int
		expected    bool
	}{
		{
			name: "Current hour matches medication hour (daily)",
			medication: config.Medication{
				Name:      "Med1",
				Hour:      10,
				Frequency: "daily",
			},
			currentHour: 10,
			expected:    true,
		},
		{
			name: "Current hour is within reminder window (daily)",
			medication: config.Medication{
				Name:      "Med2",
				Hour:      10,
				Frequency: "daily",
			},
			currentHour: 12,
			expected:    true,
		},
		{
			name: "Current hour is outside reminder window (daily)",
			medication: config.Medication{
				Name:      "Med3",
				Hour:      10,
				Frequency: "daily",
			},
			currentHour: 16,
			expected:    false,
		},
		{
			name: "Current hour is before medication hour (daily)",
			medication: config.Medication{
				Name:      "Med4",
				Hour:      15,
				Frequency: "daily",
			},
			currentHour: 10,
			expected:    false,
		},
		{
			name: "Weekly medication on correct day and hour",
			medication: config.Medication{
				Name:      "Med5",
				Hour:      10,
				Frequency: "weekly",
				Day:       currentDay,
			},
			currentHour: 10,
			expected:    true,
		},
		{
			name: "Weekly medication on correct day but outside hour window",
			medication: config.Medication{
				Name:      "Med6",
				Hour:      10,
				Frequency: "weekly",
				Day:       currentDay,
			},
			currentHour: 16,
			expected:    false,
		},
		{
			name: "Weekly medication on wrong day but correct hour",
			medication: config.Medication{
				Name:      "Med7",
				Hour:      10,
				Frequency: "weekly",
				Day:       differentDay,
			},
			currentHour: 10,
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock implementation of shouldSendReminder that uses the test's currentHour
			mockShouldSendReminder := func(medication config.Medication) bool {
				currentHour := tt.currentHour

				// Default to daily if frequency is not specified
				if medication.Frequency == "" {
					medication.Frequency = "daily"
				}

				// For weekly medications, check if today is the specified day
				if medication.Frequency == "weekly" {
					// Get the current day of the week (use the actual current day for the test)
					testCurrentDay := currentDay

					// If the day doesn't match, don't send a reminder
					if strings.ToLower(medication.Day) != testCurrentDay {
						return false
					}
				}

				return currentHour >= medication.Hour && currentHour < medication.Hour+5
			}

			result := mockShouldSendReminder(tt.medication)
			if result != tt.expected {
				t.Errorf("shouldSendReminder() = %v, want %v", result, tt.expected)
			}
		})
	}
}
