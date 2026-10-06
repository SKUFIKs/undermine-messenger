APP_NAME = undermine
MAIN_PATH = ./cmd/client/main.go
BUILD_DIR = ./bin

.PHONY: all build run clean

all: build

build:
	@echo "--> Building $(APP_NAME)..."
	mkdir -p $(BUILD_DIR)
	go build -ldflags="-s -w" -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PATH)

run:
	@echo "--> Running Fyne UI..."
	go run $(MAIN_PATH)

clean:
	rm -rf $(BUILD_DIR)