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
.PHONY: all clean deps mgmtBotMatrix mgmtBotMatrix mgmtApi pdnsBackend

# Default: clean → update deps → build everything
all: clean deps mgmtBotMatrix mgmtBotDiscord mgmtApi pdnsBackend

# Pull in module deps + upgrades
deps:
	go mod tidy
	go get -u ./...

# Ensure bin/ exists
$(BIN_DIR):
	$(MKDIRBIN)

# Build rules (uses forward-slashes)
mgmtBotMatrix: $(BIN_DIR)
	go build -o $(BIN_DIR)$(SEP)$@ src/mgmtBotMatrix/mgmtBotMatrix.go

mgmtBotDiscord: $(BIN_DIR)
	go build -o $(BIN_DIR)$(SEP)$@ src/mgmtBotDiscord/mgmtBotDiscord.go

mgmtApi: $(BIN_DIR)
	go build -o $(BIN_DIR)$(SEP)$@ src/mgmtApi/mgmtApi.go

pdnsBackend: $(BIN_DIR)
	go build -o $(BIN_DIR)$(SEP)$@ src/pdnsBackend/pdnsBackend.go

# Clean workspace
clean:
	-$(DELBIN)
