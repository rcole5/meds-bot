package reminder

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"meds-bot/internal/config"
	"meds-bot/internal/db"
	"meds-bot/internal/discord"
)

// ServiceInterface defines the interface for the reminder service
type ServiceInterface interface {
	Start(ctx context.Context) error
	Stop()
}

type Service struct {
	config   *config.Config
	store    db.StoreInterface
	discord  discord.ClientInterface
	stopCh   chan struct{}
	stopOnce sync.Once
	wg       sync.WaitGroup
}

func NewService(cfg *config.Config, store db.StoreInterface, discord discord.ClientInterface) *Service {
	return &Service{
		config:  cfg,
		store:   store,
		discord: discord,
		stopCh:  make(chan struct{}),
	}
}

// Start starts the reminder service
func (s *Service) Start(ctx context.Context) error {
	s.discord.RegisterMedicationHandler(ctx)

	s.wg.Add(1)
	go s.reminderLoop(ctx)

	log.Println("Reminder service started")
	return nil
}

// Stop stops the reminder service
func (s *Service) Stop() {
	s.stopOnce.Do(func() {
		close(s.stopCh)
		s.wg.Wait()
		log.Println("Reminder service stopped")
	})
}

// reminderLoop is the main reminder loop
func (s *Service) reminderLoop(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.GetReminderInterval())
	defer ticker.Stop()

	// Check immediately on startup
	if err := s.checkAndSendReminders(ctx); err != nil {
		log.Printf("Error checking and sending reminders: %v", err)
	}

	for {
		select {
		case <-ticker.C:
			if err := s.checkAndSendReminders(ctx); err != nil {
				log.Printf("Error checking and sending reminders: %v", err)
			}
		case <-s.stopCh:
			log.Println("Reminder loop stopped")
			return
		case <-ctx.Done():
			log.Println("Context cancelled, stopping reminder loop")
			return
		}
	}
}

// checkAndSendReminders checks if reminders need to be sent and sends them
func (s *Service) checkAndSendReminders(ctx context.Context) error {
	for _, medication := range s.config.Medications {
		if !s.shouldSendReminder(medication) {
			continue
		}

		reminder, err := s.store.GetTodayReminder(ctx, medication.Name)
		if err != nil {
			return fmt.Errorf("failed to get reminder for %s: %w", medication.Name, err)
		}

		if reminder.Acknowledged {
			continue
		}

		// Delete existing message
		if reminder.MessageID != "" {
			if err := s.discord.DeleteMessage(ctx, reminder.MessageID); err != nil {
				log.Printf("Error deleting previous message for %s: %v", medication.Name, err)
			}
		}

		newMessageID, err := s.discord.SendReminder(ctx, medication)
		if err != nil {
			return fmt.Errorf("failed to send reminder for %s: %w", medication.Name, err)
		}

		// Update the reminder with the new message ID
		if err := s.store.UpdateReminderStatus(ctx, reminder.ID, false, newMessageID); err != nil {
			return fmt.Errorf("failed to update reminder status for %s: %w", medication.Name, err)
		}
	}

	return nil
}

// shouldSendReminder checks if it's time to send a reminder for a specific medication
func (s *Service) shouldSendReminder(medication config.Medication) bool {
	// Get the location from the config
	loc, err := s.config.GetLocation()
	if err != nil {
		log.Printf("Error getting timezone location: %v, using UTC", err)
		loc = time.UTC
	}

	// Get the current time in the configured timezone
	now := time.Now().In(loc)
	currentHour := now.Hour()

	// Default to daily if frequency is not specified
	if medication.Frequency == "" {
		medication.Frequency = "daily"
	}

	// For weekly medications, check if today is the specified day
	if medication.Frequency == "weekly" {
		// Get the current day of the week
		currentDay := strings.ToLower(now.Weekday().String())

		// If the day doesn't match, don't send a reminder
		if strings.ToLower(medication.Day) != currentDay {
			return false
		}
	}

	// Check if it's time for this medication
	// Only send reminders if the current hour is within 5 hours of the medication hour and not before the medication hour
	return currentHour >= medication.Hour && currentHour < medication.Hour+5
}
