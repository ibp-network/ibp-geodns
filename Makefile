# Detect OS
ifeq ($(OS),Windows_NT)
  MKDIRBIN = if not exist $(BIN_DIR) mkdir $(BIN_DIR)
  DELBIN    = if exist     $(BIN_DIR) rmdir /S /Q $(BIN_DIR)
  EXT       = .exe
else
  MKDIRBIN = mkdir -p $(BIN_DIR)
  DELBIN    = rm -rf $(BIN_DIR)
  EXT       =
endif

# Constants
BIN_DIR := bin
SEP     := /

# Phony targets
.PHONY: all clean deps mgmtBotMatrix mgmtBotDiscord mgmtApi dnsApi serviceMonitor

# Default: clean → update deps → build everything
all: clean deps mgmtBotMatrix mgmtBotDiscord mgmtApi dnsApi serviceMonitor

# Pull in module deps + upgrades
deps:
	go mod tidy
	go get -u ./...

# Ensure bin/ exists
$(BIN_DIR):
	$(MKDIRBIN)

# Build rules, appending $(EXT) if on Windows
mgmtBotMatrix: $(BIN_DIR)
	go build -o $(BIN_DIR)$(SEP)mgmtBotMatrix$(EXT) src/mgmtBotMatrix/mgmtBotMatrix.go

mgmtBotDiscord: $(BIN_DIR)
	go build -o $(BIN_DIR)$(SEP)mgmtBotDiscord$(EXT) src/mgmtBotDiscord/mgmtBotDiscord.go

mgmtApi: $(BIN_DIR)
	go build -o $(BIN_DIR)$(SEP)mgmtApi$(EXT) src/mgmtApi/mgmtApi.go

dnsApi: $(BIN_DIR)
	go build -o $(BIN_DIR)$(SEP)dnsApi$(EXT) src/dnsApi/dnsApi.go

serviceMonitor: $(BIN_DIR)
	go build -o $(BIN_DIR)$(SEP)serviceMonitor$(EXT) src/serviceMonitor/serviceMonitor.go

# Clean workspace
clean:
	-$(DELBIN)
