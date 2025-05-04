# Detect OS
ifeq ($(OS),Windows_NT)
  MKDIRBIN = if not exist $(BIN_DIR) mkdir $(BIN_DIR)
  DELBIN    = if exist     $(BIN_DIR) rmdir /S /Q $(BIN_DIR)
else
  MKDIRBIN = mkdir -p $(BIN_DIR)
  DELBIN    = rm -rf $(BIN_DIR)
endif

# Constants
BIN_DIR := bin
SEP     := /

# Phony targets
.PHONY: all clean deps mgmt-matrixbot mgmt-discordbot mgmt-api powerdns-backend

# Default: clean → update deps → build everything
all: clean deps mgmt-matrixbot mgmt-discordbot mgmt-api powerdns-backend

# Pull in module deps + upgrades
deps:
	go mod tidy
	go get -u ./...

# Ensure bin/ exists
$(BIN_DIR):
	$(MKDIRBIN)

# Build rules (uses forward-slashes)
mgmt-matrixbot: $(BIN_DIR)
	go build -o $(BIN_DIR)$(SEP)$@ src/mgmt-matrixbox/mgmt-matrixbot.go

mgmt-discordbot: $(BIN_DIR)
	go build -o $(BIN_DIR)$(SEP)$@ src/mgmt-discordbot/mgmt-discordbot.go

mgmt-api: $(BIN_DIR)
	go build -o $(BIN_DIR)$(SEP)$@ src/mgmt-api/mgmt-api.go

powerdns-backend: $(BIN_DIR)
	go build -o $(BIN_DIR)$(SEP)$@ src/powerdns-backend/powerdns-backend.go

# Clean workspace
clean:
	-$(DELBIN)
