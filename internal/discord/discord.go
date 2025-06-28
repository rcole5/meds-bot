package discord

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
	"meds-bot/internal/config"
	"meds-bot/internal/db"
)

// ClientInterface defines the interface for Discord operations
type ClientInterface interface {
	Close() error
	SendReminder(ctx context.Context, medication config.Medication) (string, error)
	DeleteMessage(ctx context.Context, messageID string) error
	RegisterMedicationHandler(ctx context.Context)
}

type Client struct {
	session       *discordgo.Session
	channelID     string
	userIDToPing  string
	store         db.StoreInterface
	handlersMutex sync.Mutex
	handlers      map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate)
}

// NewClient creates a new Discord client
func NewClient(ctx context.Context, cfg *config.Config, store db.StoreInterface) (*Client, error) {
	session, err := discordgo.New("Bot " + cfg.DiscordToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create Discord session: %w", err)
	}

	client := &Client{
		session:      session,
		channelID:    cfg.DiscordChannelID,
		userIDToPing: cfg.DiscordUserIDToPing,
		store:        store,
		handlers:     make(map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate)),
	}

	session.AddHandler(client.handleInteraction)

	if err := session.Open(); err != nil {
		return nil, fmt.Errorf("failed to open Discord connection: %w", err)
	}

	return client, nil
}

// Close closes the Discord session
func (c *Client) Close() error {
	return c.session.Close()
}

// SendReminder sends a reminder message with a button
func (c *Client) SendReminder(ctx context.Context, medication config.Medication) (string, error) {
	// Create a unique custom ID for the button
	customID := fmt.Sprintf("medication_taken_%s", medication.Name)

	// Create the button component
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    fmt.Sprintf("I took %s", medication.Name),
					Style:    discordgo.SuccessButton,
					CustomID: customID,
					Emoji: &discordgo.ComponentEmoji{
						Name: "✅",
					},
				},
			},
		},
	}

	content := ""
	if c.userIDToPing != "" {
		content += fmt.Sprintf("<@%s> ", c.userIDToPing)
	}
	content += fmt.Sprintf("🔔 **Medication Reminder: %s** 🔔\n", medication.Name)
	content += fmt.Sprintf("It's time to take your %s! Please click the button below once you've taken it.", medication.Name)

	msg, err := c.session.ChannelMessageSendComplex(c.channelID, &discordgo.MessageSend{
		Content:    content,
		Components: components,
	})

	if err != nil {
		return "", fmt.Errorf("failed to send reminder message: %w", err)
	}

	return msg.ID, nil
}

// DeleteMessage deletes a message
func (c *Client) DeleteMessage(ctx context.Context, messageID string) error {
	if messageID == "" {
		return nil
	}

	err := c.session.ChannelMessageDelete(c.channelID, messageID)
	if err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}

	return nil
}

// RegisterHandler registers a handler for a custom ID prefix
func (c *Client) RegisterHandler(prefix string, handler func(s *discordgo.Session, i *discordgo.InteractionCreate)) {
	c.handlersMutex.Lock()
	defer c.handlersMutex.Unlock()
	c.handlers[prefix] = handler
}

// handleInteraction handles all interactions
func (c *Client) handleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Only handle message component interactions (buttons)
	if i.Type != discordgo.InteractionMessageComponent {
		return
	}

	customID := i.MessageComponentData().CustomID

	c.handlersMutex.Lock()
	defer c.handlersMutex.Unlock()

	// Find a handler for this custom ID
	for prefix, handler := range c.handlers {
		if strings.HasPrefix(customID, prefix) {
			handler(s, i)
			return
		}
	}

	log.Printf("Warning: No handler found for custom ID: %s", customID)
}

// RegisterMedicationHandler registers the handler for medication buttons
func (c *Client) RegisterMedicationHandler(ctx context.Context) {
	c.RegisterHandler("medication_taken_", func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		customID := i.MessageComponentData().CustomID

		// Parse the medication name from the customID
		if len(customID) <= 17 {
			log.Printf("Invalid customID format: %s", customID)
			return
		}

		// Get everything after "medication_taken_"
		medicationName := customID[17:]

		reminder, err := c.store.GetTodayReminder(ctx, medicationName)
		if err != nil {
			log.Printf("Error getting reminder for %s: %v", medicationName, err)
			c.respondWithError(s, i, fmt.Sprintf("Error getting reminder: %v", err))
			return
		}

		// If already acknowledged, just respond
		if reminder.Acknowledged {
			err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("You've already acknowledged taking your %s today. Thank you!", medicationName),
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			if err != nil {
				log.Printf("Error responding to interaction for %s: %v", medicationName, err)
			}
			return
		}

		err = c.store.UpdateReminderStatus(ctx, reminder.ID, true, i.Message.ID)
		if err != nil {
			log.Printf("Error updating reminder for %s: %v", medicationName, err)
			c.respondWithError(s, i, fmt.Sprintf("Error updating reminder: %v", err))
			return
		}

		// Update the original message
		content := fmt.Sprintf("✅ **%s Taken** ✅\nThank you for taking your %s today!", medicationName, medicationName)

		// Remove the button by setting empty components and update the message content
		_, err = s.ChannelMessageEditComplex(&discordgo.MessageEdit{
			Channel:    c.channelID,
			ID:         i.Message.ID,
			Content:    &content,
			Components: &[]discordgo.MessageComponent{},
		})
		if err != nil {
			log.Printf("Error updating message for %s: %v", medicationName, err)
		}

		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Thank you for taking your %s! Your response has been recorded.", medicationName),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		if err != nil {
			log.Printf("Error responding to interaction for %s: %v", medicationName, err)
		}
	})
}

// respondWithError responds to an interaction with an error message
func (c *Client) respondWithError(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Error: %s", message),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Printf("Error responding with error message: %v", err)
	}
}
