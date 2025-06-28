# Medication Reminder Discord Bot

A Discord bot that sends medication reminders at specified times and tracks acknowledgments.

## Features

- Sends reminders for medications at configured times
- Supports both daily and weekly medication schedules
- Allows users to acknowledge taking medications via a button click
- Continues to send reminders every configured interval until acknowledged
- Supports multiple medications with different schedules
- Pings a specific user in reminder messages (optional)
- Graceful shutdown with proper resource cleanup

## Project Structure

The project follows a clean architecture with separation of concerns:

- `internal/config`: Configuration loading and validation
- `internal/db`: Database operations for tracking reminders
- `internal/discord`: Discord API interactions
- `internal/reminder`: Reminder scheduling and management
- `main.go`: Application entry point

## Setup

### Prerequisites

- Go 1.18 or higher
- A Discord bot token (create one at the [Discord Developer Portal](https://discord.com/developers/applications))
- A Discord server where the bot is invited with proper permissions

### Installation

1. Clone the repository:
   ```
   git clone https://github.com/rcole5/meds-bot.git
   cd meds-bot
   ```

2. Copy the example `.env` file and edit it with your configuration:
   ```
   cp .env.example .env
   ```

3. Edit the `.env` file with your Discord bot token, channel ID, and other settings.

4. Build the application:
   ```
   go build
   ```

5. Run the bot:
   ```
   ./meds-bot
   ```

## Configuration

All configuration is done through environment variables, which can be set in the `.env` file:

### Discord Configuration

- `DISCORD_TOKEN`: Your Discord bot token
- `DISCORD_CHANNEL_ID`: The ID of the channel where reminders will be posted
- `DISCORD_USER_ID_TO_PING`: (Optional) The ID of the user to ping in reminder messages

### Reminder Configuration

- `REMINDER_INTERVAL_MINUTES`: How often to check and send reminders (in minutes)
- `DB_PATH`: (Optional) Path to the SQLite database file (defaults to `./meds_reminder.db`)

### Medication Configuration

You can configure multiple medications by adding numbered environment variables:

- `MED_1_NAME`: Name of the first medication
- `MED_1_HOUR`: Hour to send the reminder (24-hour format, 0-23)
- `MED_1_FREQUENCY`: (Optional) Frequency of the reminder - either "daily" (default) or "weekly"
- `MED_1_DAY`: (Required for weekly frequency) Day of the week to send the reminder (e.g., "monday", "tuesday", etc.)
- `MED_2_NAME`: Name of the second medication
- `MED_2_HOUR`: Hour to send the reminder for the second medication
- `MED_2_FREQUENCY`: (Optional) Frequency of the second medication
- `MED_2_DAY`: (Required for weekly frequency) Day of the week for the second medication
- And so on...

## How It Works

1. The bot starts and loads configuration from environment variables
2. It connects to Discord and initializes the database
3. For each configured medication, it checks if it's time to send a reminder
4. If it's time and the medication hasn't been acknowledged today, it sends a reminder message with a button
5. When a user clicks the button, the bot marks the medication as acknowledged for the day
6. The bot continues to check and send reminders at the configured interval

## Deployment Options

### Local Deployment

For local deployment, follow the [Setup](#setup) instructions above.

### Docker Deployment

A Dockerfile is provided to containerize the application:

```bash
# Build the Docker image
docker build -t meds-bot:latest .

# Run the container
docker run -v $(pwd)/data:/app/data --env-file .env meds-bot:latest
```

### Kubernetes (k3s) Deployment

For deploying to a Kubernetes cluster (specifically k3s), configuration files are provided in the `k8s` directory.

See the [k3s Deployment Guide](k8s/README.md) for detailed instructions.

## Development

### Building from Source

```
go build
```

### Running Tests

```
go test ./...
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.
