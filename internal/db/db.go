package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

// StoreInterface defines the interface for database operations
type StoreInterface interface {
	Close() error
	GetTodayReminder(ctx context.Context, medicationType string) (*Reminder, error)
	UpdateReminderStatus(ctx context.Context, id int64, acknowledged bool, messageID string) error
}

type Store struct {
	db *sql.DB
}

type Reminder struct {
	ID               int64
	Date             string
	MedicationType   string
	Acknowledged     bool
	LastReminderTime time.Time
	MessageID        string
}

// NewStore creates a new database store
func NewStore(ctx context.Context, dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	// Verify connection
	ctxPing, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctxPing); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	store := &Store{db: db}

	if err := store.initSchema(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize database schema: %w", err)
	}

	return store, nil
}

// Close closes the database connection
func (s *Store) Close() error {
	return s.db.Close()
}

// initSchema initializes the database schema
func (s *Store) initSchema(ctx context.Context) error {
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS reminders (
		id INTEGER PRIMARY KEY,
		date TEXT NOT NULL,
		medication_type TEXT NOT NULL,
		acknowledged INTEGER DEFAULT 0,
		last_reminder_time TEXT,
		message_id TEXT
	);`

	ctxExec, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := s.db.ExecContext(ctxExec, createTableSQL)
	return err
}

// GetTodayReminder gets or creates a reminder for today for a specific medication
func (s *Store) GetTodayReminder(ctx context.Context, medicationType string) (*Reminder, error) {
	today := time.Now().Format("2006-01-02")

	ctxQuery, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var id int64
	var acknowledged int
	var messageID sql.NullString
	var lastReminderTimeStr sql.NullString

	err := s.db.QueryRowContext(ctxQuery, "SELECT id, acknowledged, message_id, last_reminder_time FROM reminders WHERE date = ? AND medication_type = ?", today, medicationType).Scan(&id, &acknowledged, &messageID, &lastReminderTimeStr)

	if err == nil {
		var lastReminderTime time.Time
		if lastReminderTimeStr.Valid {
			lastReminderTime, _ = time.Parse(time.RFC3339, lastReminderTimeStr.String)
		}

		return &Reminder{
			ID:               id,
			Date:             today,
			MedicationType:   medicationType,
			Acknowledged:     acknowledged == 1,
			LastReminderTime: lastReminderTime,
			MessageID:        messageID.String,
		}, nil
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to query reminder: %w", err)
	}

	ctxInsert, cancelInsert := context.WithTimeout(ctx, 5*time.Second)
	defer cancelInsert()

	result, err := s.db.ExecContext(ctxInsert,
		"INSERT INTO reminders (date, medication_type, acknowledged) VALUES (?, ?, 0)",
		today, medicationType)
	if err != nil {
		return nil, fmt.Errorf("failed to create reminder: %w", err)
	}

	id, err = result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert ID: %w", err)
	}

	return &Reminder{
		ID:             id,
		Date:           today,
		MedicationType: medicationType,
		Acknowledged:   false,
	}, nil
}

// UpdateReminderStatus updates the status of a reminder
func (s *Store) UpdateReminderStatus(ctx context.Context, id int64, acknowledged bool, messageID string) error {
	ctxUpdate, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var ack int
	if acknowledged {
		ack = 1
	}

	now := time.Now().Format(time.RFC3339)

	_, err := s.db.ExecContext(ctxUpdate,
		"UPDATE reminders SET acknowledged = ?, message_id = ?, last_reminder_time = ? WHERE id = ?",
		ack, messageID, now, id)
	if err != nil {
		return fmt.Errorf("failed to update reminder: %w", err)
	}

	return nil
}
