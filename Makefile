.PHONY: all build build-cli build-api install test clean

BINARY_NAME=project
API_BINARY_NAME=project-api
BUILD_DIR=bin

all: test build

build: build-cli build-api

build-cli:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/project

build-api:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(API_BINARY_NAME) ./cmd/api

install: build-cli
	@echo "Installing $(BINARY_NAME) to /usr/local/bin/$(BINARY_NAME)..."
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	sudo chmod 755 /usr/local/bin/$(BINARY_NAME)
	@echo "Installation complete. Run 'sudo project --help'"

test:
	go test -v ./...

clean:
	rm -rf $(BUILD_DIR)
