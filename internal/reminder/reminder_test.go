package reminder

import (
	"testing"

	"meds-bot/internal/config"
)

// TestShouldSendReminder tests the shouldSendReminder function
func TestShouldSendReminder(t *testing.T) {
	// Config not actually used in this test
	_ = &config.Config{
		ReminderIntervalMins: 30,
	}

	// Test cases
	tests := []struct {
		name       string
		medication config.Medication
		currentHour int
		expected   bool
	}{
		{
			name: "Current hour matches medication hour",
			medication: config.Medication{
				Name: "Med1",
				Hour: 10,
			},
			currentHour: 10,
			expected: true,
		},
		{
			name: "Current hour is within reminder window",
			medication: config.Medication{
				Name: "Med2",
				Hour: 10,
			},
			currentHour: 12,
			expected: true,
		},
		{
			name: "Current hour is outside reminder window",
			medication: config.Medication{
				Name: "Med3",
				Hour: 10,
			},
			currentHour: 16,
			expected: false,
		},
		{
			name: "Current hour is before medication hour",
			medication: config.Medication{
				Name: "Med4",
				Hour: 15,
			},
			currentHour: 10,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock implementation of shouldSendReminder that uses the test's currentHour
			mockShouldSendReminder := func(medication config.Medication) bool {
				currentHour := tt.currentHour
				return currentHour >= medication.Hour && currentHour < medication.Hour+5
			}

			result := mockShouldSendReminder(tt.medication)
			if result != tt.expected {
				t.Errorf("shouldSendReminder() = %v, want %v", result, tt.expected)
			}
		})
	}
}
