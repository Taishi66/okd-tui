APP_NAME := okd-tui
VERSION := 0.1.0
BUILD_DIR := bin
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: build run clean test

build:
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) ./cmd/

run: build
	./$(BUILD_DIR)/$(APP_NAME)

clean:
	rm -rf $(BUILD_DIR)

test:
	go test ./...

deps:
	go mod tidy

lint:
	golangci-lint run ./...
