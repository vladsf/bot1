# Makefile for Telegram Bot
include .env

BOT_NAME = bot1
GO_FILES = cmd/bot/*.go
VERSION = $(shell git describe --tags --always 2>/dev/null || echo "v0.0.0-dev") # Get version from Git tags

# Build the bot binary
build:
	go build -ldflags="-X main.Version=$(VERSION)" -o $(BOT_NAME) $(GO_FILES)

# Run the bot (for development)
run: build
	@export TG_BOT_TOKEN=$(TG_BOT_TOKEN) \
		IMAGE_PROCESSING_SERVER_URL=$(IMAGE_PROCESSING_SERVER_URL) \
		IMAGE_PROCESSING_API_TOKEN=$(IMAGE_PROCESSING_API_TOKEN) \
		&& ./$(BOT_NAME)

# Build and run in one step
dev: build run

# Clean up the binary
clean:
	rm -f $(BOT_NAME)

# Format the Go code
fmt:
	go fmt $(GO_FILES)

# Run tests (if you have any)
test:
	go test ./...

# Build a Docker image (optional - for deployment)
docker:
	docker build -t vladsf/$(BOT_NAME) .

# Push the Docker image (optional)
docker-push: docker
	docker push vladsf/$(BOT_NAME)

# Deploy the bot binary (optional)
deploy: build
	@echo "Starting deployment at $(shell date '+%Y-%m-%d %H:%M:%S')"
	@echo "Stopping $(BOT_NAME) service"
	@ssh $(BOT_REMOTE_HOST) systemctl stop $(BOT_NAME).service
	rsync -avP ./$(BOT_NAME) $(BOT_REMOTE_HOST):$(BOT_REMOTE_DIR) || { \
        	echo "rsync failed with exit code $$?"; \
        	exit 1; \
        }
	@echo "Starting $(BOT_NAME) service"
	@ssh $(BOT_REMOTE_HOST) systemctl start $(BOT_NAME).service
	@ssh $(BOT_REMOTE_HOST) systemctl status $(BOT_NAME).service
	@echo "Deployment finished at $(shell date '+%Y-%m-%d %H:%M:%S')"

.PHONY: build run dev clean fmt test docker docker-push deploy # Mark targets as phony
