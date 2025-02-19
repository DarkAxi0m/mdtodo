APP_NAME := mdtodo
INSTALL_DIR := /usr/local/bin
SRC_DIR := .
BUILD_DIR := ./bin
BIN := $(BUILD_DIR)/$(APP_NAME)

.PHONY: all build install clean

all: build

build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BIN) $(SRC_DIR)

install: build
	@echo "Installing $(APP_NAME)..."
	@sudo install -m 755 $(BIN) $(INSTALL_DIR)/$(APP_NAME)
	@echo "$(APP_NAME) installed to $(INSTALL_DIR)"

clean:
	@echo "Cleaning up..."
	@rm -rf $(BUILD_DIR)
