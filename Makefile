# Detect OS
ifeq ($(OS),Windows_NT)
    DETECTED_OS := Windows
    SHELL := cmd.exe
    RM := del /Q
    RMDIR := rmdir /S /Q
    MKDIR := mkdir
    CP := xcopy /E /I /Y
    CPFILE := copy /Y
    SEP := \\
    EXE := .exe
    NPM := npm.cmd
    NULL := nul
else
    DETECTED_OS := $(shell uname -s)
    RM := rm -f
    RMDIR := rm -rf
    MKDIR := mkdir -p
    CP := cp -r
    CPFILE := cp
    SEP := /
    EXE :=
    NPM := npm
    NULL := /dev/null
endif

# Variables
GO := go
GOFLAGS := -v
BINDIR := bin
SRCDIR := src
DASHDIR := $(SRCDIR)$(SEP)IBDash
DASHBUILD := $(DASHDIR)$(SEP)build
PUBLICDIR := public
HTACCESS := $(PUBLICDIR)$(SEP).htaccess

# Go binaries to build
GO_BINARIES := IBPDns IBPMonitor IBPCollator
GO_SOURCES := $(foreach bin,$(GO_BINARIES),$(SRCDIR)$(SEP)$(bin)$(SEP)$(bin).go)

# Default target
.PHONY: all
all: clean go-build dash-build copy-public

# Clean build artifacts
.PHONY: clean
clean:
	@echo Cleaning build artifacts...
ifeq ($(DETECTED_OS),Windows)
	@if exist $(BINDIR) $(RMDIR) $(BINDIR) 2>$(NULL)
	@if exist $(DASHBUILD) $(RMDIR) $(DASHBUILD) 2>$(NULL)
	@if exist $(PUBLICDIR) $(RMDIR) $(PUBLICDIR) 2>$(NULL)
else
	@$(RMDIR) $(BINDIR) 2>$(NULL) || true
	@$(RMDIR) $(DASHBUILD) 2>$(NULL) || true
	@$(RMDIR) $(PUBLICDIR) 2>$(NULL) || true
endif

# Create directories
.PHONY: dirs
dirs:
	@echo Creating directories...
ifeq ($(DETECTED_OS),Windows)
	@if not exist $(BINDIR) $(MKDIR) $(BINDIR)
	@if not exist $(PUBLICDIR) $(MKDIR) $(PUBLICDIR)
else
	@$(MKDIR) $(BINDIR)
	@$(MKDIR) $(PUBLICDIR)
endif

# Build Go binaries
.PHONY: go-build
go-build: dirs
	@echo Building Go binaries for $(DETECTED_OS)...
	@$(foreach bin,$(GO_BINARIES),\
		echo Building $(bin)... && \
		$(GO) build $(GOFLAGS) -o $(BINDIR)$(SEP)$(bin)$(EXE) .$(SEP)$(SRCDIR)$(SEP)$(bin)$(SEP)$(bin).go &&) true

# Install npm dependencies for dashboard
.PHONY: dash-deps
dash-deps:
	@echo Installing dashboard dependencies...
ifeq ($(DETECTED_OS),Windows)
	@cd $(DASHDIR) && $(NPM) install
else
	@cd $(DASHDIR) && $(NPM) install
endif

# Build React dashboard
.PHONY: dash-build
dash-build: dash-deps
	@echo Building React dashboard...
ifeq ($(DETECTED_OS),Windows)
	@cd $(DASHDIR) && $(NPM) run build
else
	@cd $(DASHDIR) && $(NPM) run build
endif

# Copy dashboard build to public directory
.PHONY: copy-public
copy-public: dirs create-htaccess
	@echo Copying dashboard to public directory...
ifeq ($(DETECTED_OS),Windows)
	@if exist $(DASHBUILD) $(CP) $(DASHBUILD) $(PUBLICDIR)
else
	@if [ -d $(DASHBUILD) ]; then $(CP) $(DASHBUILD)/* $(PUBLICDIR)/; fi
endif

# Create .htaccess file
.PHONY: create-htaccess
create-htaccess: dirs
	@echo Creating .htaccess file...
ifeq ($(DETECTED_OS),Windows)
	@echo RewriteEngine On > $(HTACCESS)
	@echo RewriteBase / >> $(HTACCESS)
	@echo RewriteCond %%{REQUEST_FILENAME} !-f >> $(HTACCESS)
	@echo RewriteCond %%{REQUEST_FILENAME} !-d >> $(HTACCESS)
	@echo RewriteRule . /index.html [L] >> $(HTACCESS)
else
	@echo "RewriteEngine On" > $(HTACCESS)
	@echo "RewriteBase /" >> $(HTACCESS)
	@echo "RewriteCond %{REQUEST_FILENAME} !-f" >> $(HTACCESS)
	@echo "RewriteCond %{REQUEST_FILENAME} !-d" >> $(HTACCESS)
	@echo "RewriteRule . /index.html [L]" >> $(HTACCESS)
endif

# Development targets
.PHONY: dev
dev: go-build
	@echo Starting development mode...
	@echo Go binaries built. Start React dev server manually with: cd $(DASHDIR) && npm start

# Build only Go binaries
.PHONY: go
go: go-build

# Build only dashboard
.PHONY: dash
dash: dash-build copy-public

# Run specific services
.PHONY: run-dns
run-dns: go-build
	@echo Starting IBPDns...
	@$(BINDIR)$(SEP)IBPDns$(EXE) -config config$(SEP)ibpdns.json

.PHONY: run-monitor
run-monitor: go-build
	@echo Starting IBPMonitor...
	@$(BINDIR)$(SEP)IBPMonitor$(EXE) -config config$(SEP)ibpmonitor.json

.PHONY: run-collator
run-collator: go-build
	@echo Starting IBPCollator...
	@$(BINDIR)$(SEP)IBPCollator$(EXE) -config config$(SEP)ibpcollator.json

# Help
.PHONY: help
help:
	@echo Available targets:
	@echo   all          - Build everything (default)
	@echo   clean        - Remove all build artifacts
	@echo   go-build     - Build Go binaries only
	@echo   dash-build   - Build React dashboard only
	@echo   dash-deps    - Install dashboard npm dependencies
	@echo   copy-public  - Copy dashboard build to public directory
	@echo   dev          - Build for development
	@echo   run-dns      - Build and run IBPDns
	@echo   run-monitor  - Build and run IBPMonitor
	@echo   run-collator - Build and run IBPCollator
	@echo   help         - Show this help message
	@echo.
	@echo Detected OS: $(DETECTED_OS)

# Install all dependencies (Go and npm)
.PHONY: deps
deps: dash-deps
	@echo Installing Go dependencies...
	@$(GO) mod download
	@$(GO) mod tidy

# Build for production with optimizations
.PHONY: prod
prod: clean
	@echo Building for production...
	$(eval GOFLAGS := -v -ldflags="-s -w")
	$(eval export NODE_ENV=production)
	@$(MAKE) all

# Test targets
.PHONY: test
test:
	@echo Running Go tests...
	@$(GO) test ./...

.PHONY: test-dash
test-dash:
	@echo Running dashboard tests...
ifeq ($(DETECTED_OS),Windows)
	@cd $(DASHDIR) && $(NPM) test
else
	@cd $(DASHDIR) && $(NPM) test
endif