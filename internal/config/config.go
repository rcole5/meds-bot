package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// ConfigSource represents the source of configuration
type ConfigSource string

const (
	EnvSource   ConfigSource = "env"
	JSONSource  ConfigSource = "json"
	DefaultPath              = "./config.json"
)

type Config struct {
	DiscordToken         string
	DiscordChannelID     string
	DiscordUserIDToPing  string
	ReminderIntervalMins int
	Medications          []Medication
	DBPath               string
	Timezone             string
}

type Medication struct {
	Name      string
	Hour      int
	Frequency string
	Day       string
}

// LoadConfig loads the application configuration from environment variables by default
func LoadConfig() (*Config, error) {
	// Try to determine config source from CONFIG_SOURCE env var
	configSource := os.Getenv("CONFIG_SOURCE")
	if configSource == "" {
		configSource = string(EnvSource)
	}

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = DefaultPath
	}

	switch strings.ToLower(configSource) {
	case string(JSONSource):
		return LoadJSONConfig(configPath)
	default:
		return LoadEnvConfig()
	}
}

// LoadJSONConfig loads configuration from a JSON file
func LoadJSONConfig(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse JSON config: %w", err)
	}

	// Validate the config
	if err := validateConfig(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// validateConfig validates the configuration
func validateConfig(cfg *Config) error {
	if cfg.DiscordToken == "" {
		return fmt.Errorf("Discord token is required")
	}

	if cfg.DiscordChannelID == "" {
		return fmt.Errorf("Discord channel ID is required")
	}

	if cfg.ReminderIntervalMins < 1 {
		return fmt.Errorf("reminder interval must be at least 1 minute")
	}

	if len(cfg.Medications) == 0 {
		return fmt.Errorf("at least one medication is required")
	}

	for i, med := range cfg.Medications {
		if med.Name == "" {
			return fmt.Errorf("medication #%d has no name", i+1)
		}
		if med.Hour < 0 || med.Hour > 23 {
			return fmt.Errorf("medication %s has invalid hour: %d (must be between 0 and 23)", med.Name, med.Hour)
		}

		// Validate frequency
		if med.Frequency == "" {
			med.Frequency = "daily" // Default to daily if not specified
		} else if med.Frequency != "daily" && med.Frequency != "weekly" {
			return fmt.Errorf("medication %s has invalid frequency: %s (must be 'daily' or 'weekly')", med.Name, med.Frequency)
		}

		// Validate day for weekly medications
		if med.Frequency == "weekly" && med.Day == "" {
			return fmt.Errorf("medication %s has weekly frequency but no day specified", med.Name)
		}
	}

	if cfg.DBPath == "" {
		cfg.DBPath = "./meds_reminder.db"
	}

	// Validate and set default timezone
	if cfg.Timezone == "" {
		cfg.Timezone = "UTC"
	} else {
		// Check if the timezone is valid
		_, err := time.LoadLocation(cfg.Timezone)
		if err != nil {
			return fmt.Errorf("invalid timezone: %s - %w", cfg.Timezone, err)
		}
	}

	return nil
}

// LoadEnvConfig loads configuration from environment variables
func LoadEnvConfig() (*Config, error) {
	err := godotenv.Load()
	if err != nil {
		// Only log a warning, don't fail if .env file doesn't exist
		// This allows using environment variables without a .env file
		log.Printf("Warning: Error loading .env file: %v\n", err)
	}

	token := os.Getenv("DISCORD_TOKEN")
	channelID := os.Getenv("DISCORD_CHANNEL_ID")
	userIDToPing := os.Getenv("DISCORD_USER_ID_TO_PING")

	intervalStr := os.Getenv("REMINDER_INTERVAL_MINUTES")
	interval := 30
	if intervalStr != "" {
		parsedInterval, err := strconv.Atoi(intervalStr)
		if err != nil {
			return nil, fmt.Errorf("invalid REMINDER_INTERVAL_MINUTES: %w", err)
		}
		interval = parsedInterval
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./meds_reminder.db"
	}

	timezone := os.Getenv("TIMEZONE")
	if timezone == "" {
		timezone = "UTC" // Default to UTC if not specified
	}

	var medications []Medication

	// Dynamically load all medications from environment variables
	for i := 1; ; i++ {
		nameKey := fmt.Sprintf("MED_%d_NAME", i)
		name := os.Getenv(nameKey)

		// Exit case, no env found
		if name == "" {
			break
		}

		hourKey := fmt.Sprintf("MED_%d_HOUR", i)
		hourStr := os.Getenv(hourKey)
		var hour int
		if hourStr != "" {
			parsedHour, err := strconv.Atoi(hourStr)
			if err != nil {
				return nil, fmt.Errorf("invalid %s: %w", hourKey, err)
			}
			hour = parsedHour
		} else {
			log.Printf("No hour found for %s, skipping this medication.\n", name)
			continue
		}

		// Get frequency (default to "daily" if not specified)
		frequencyKey := fmt.Sprintf("MED_%d_FREQUENCY", i)
		frequency := os.Getenv(frequencyKey)
		if frequency == "" {
			frequency = "daily"
		}

		// Get day (only needed for weekly frequency)
		day := ""
		if frequency == "weekly" {
			dayKey := fmt.Sprintf("MED_%d_DAY", i)
			day = os.Getenv(dayKey)
		}

		// Add the medication to our list
		medications = append(medications, Medication{
			Name:      name,
			Hour:      hour,
			Frequency: frequency,
			Day:       day,
		})

		log.Printf("Loaded medication: %s, hour: %d, frequency: %s, day: %s\n", name, hour, frequency, day)
	}

	config := &Config{
		DiscordToken:         token,
		DiscordChannelID:     channelID,
		DiscordUserIDToPing:  userIDToPing,
		ReminderIntervalMins: interval,
		Medications:          medications,
		DBPath:               dbPath,
		Timezone:             timezone,
	}

	// Validate the config
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	medicationCount := len(medications)
	log.Printf("Loaded %d medications from environment variables\n", medicationCount)

	return config, nil
}

// GetReminderInterval returns the reminder interval as a time.Duration
func (c *Config) GetReminderInterval() time.Duration {
	return time.Duration(c.ReminderIntervalMins) * time.Minute
}

// GetLocation returns the time.Location for the configured timezone
func (c *Config) GetLocation() (*time.Location, error) {
	return time.LoadLocation(c.Timezone)
}
