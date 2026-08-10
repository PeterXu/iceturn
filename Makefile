.PHONY: build run test clean deps fmt lint build-all docker-build docker-up docker-down docker-logs

# Binary name
BINARY_NAME=icecoordinator

# Build directory
BUILD_DIR=./bin

# Main package
MAIN_PACKAGE=./cmd/icecoordinator

build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)

run: build
	$(BUILD_DIR)/$(BINARY_NAME)

test:
	go test -v ./...

clean:
	rm -rf $(BUILD_DIR)
	go clean

deps:
	go mod download
	go mod tidy

# Development helpers
fmt:
	go fmt ./...

lint:
	golangci-lint run

# Build for multiple platforms
build-all:
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PACKAGE)
	GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PACKAGE)
	GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PACKAGE)

# Docker commands
docker-build:
	docker build -t iceturn:latest .

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f

docker-restart:
	docker-compose restart

# Quick deploy
docker-deploy:
	./docker-deploy.sh