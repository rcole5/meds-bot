package db

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestGetTodayReminder(t *testing.T) {
	// Create a temporary database file
	dbPath := "test_reminder.db"
	defer os.Remove(dbPath)

	// Create a new store
	ctx := context.Background()
	store, err := NewStore(ctx, dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Test case: Get a reminder for a medication that doesn't exist yet
	medicationType := "TestMed"
	reminder, err := store.GetTodayReminder(ctx, medicationType)
	if err != nil {
		t.Fatalf("Failed to get reminder: %v", err)
	}

	// Verify the reminder was created with the correct values
	if reminder.MedicationType != medicationType {
		t.Errorf("Expected medication type %s, got %s", medicationType, reminder.MedicationType)
	}
	if reminder.Acknowledged {
		t.Errorf("Expected reminder to not be acknowledged")
	}
	if reminder.Date != time.Now().Format("2006-01-02") {
		t.Errorf("Expected date %s, got %s", time.Now().Format("2006-01-02"), reminder.Date)
	}

	// Test case: Get the same reminder again, should return the existing one
	reminder2, err := store.GetTodayReminder(ctx, medicationType)
	if err != nil {
		t.Fatalf("Failed to get reminder second time: %v", err)
	}

	// Verify it's the same reminder
	if reminder.ID != reminder2.ID {
		t.Errorf("Expected same reminder ID, got %d and %d", reminder.ID, reminder2.ID)
	}

	// Test case: Update the reminder status
	err = store.UpdateReminderStatus(ctx, reminder.ID, true, "test-message-id")
	if err != nil {
		t.Fatalf("Failed to update reminder status: %v", err)
	}

	// Get the reminder again and verify the status was updated
	reminder3, err := store.GetTodayReminder(ctx, medicationType)
	if err != nil {
		t.Fatalf("Failed to get reminder after update: %v", err)
	}

	if !reminder3.Acknowledged {
		t.Errorf("Expected reminder to be acknowledged")
	}
	if reminder3.MessageID != "test-message-id" {
		t.Errorf("Expected message ID 'test-message-id', got %s", reminder3.MessageID)
	}
}