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
.PHONY: all clean deps IBPDns IBPMonitor IBPCollator

# Default: clean → update deps → build everything
all: clean deps IBPDns IBPMonitor IBPCollator

# Pull in module deps + upgrades
deps:
	go mod tidy
	go get -u ./...

# Ensure bin/ exists
$(BIN_DIR):
	$(MKDIRBIN)

# Build rules, appending $(EXT) if on Windows
IBPDns: $(BIN_DIR)
	go build -o $(BIN_DIR)$(SEP)IBPDns$(EXT) src/IBPDns/IBPDns.go

IBPMonitor: $(BIN_DIR)
	go build -o $(BIN_DIR)$(SEP)IBPMonitor$(EXT) src/IBPMonitor/IBPMonitor.go

IBPCollator: $(BIN_DIR)
	go build -o $(BIN_DIR)$(SEP)IBPCollator$(EXT) src/IBPCollator/IBPCollator.go

# Clean workspace
clean:
	-$(DELBIN)
