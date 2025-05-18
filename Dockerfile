FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -a -o meds-bot .

FROM alpine:latest

RUN apk add --no-cache ca-certificates libc6-compat curl

WORKDIR /app

COPY --from=builder /app/meds-bot .

RUN mkdir -p /app/data

ENV DB_PATH=/app/data/meds_reminder.db

# Expose the health check port
EXPOSE 8080

# Add health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/health || exit 1

RUN adduser -D -u 1000 appuser
RUN chown -R appuser:appuser /app
USER appuser

CMD ["./meds-bot"]
