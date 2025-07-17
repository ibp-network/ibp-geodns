# Detect OS
ifeq ($(OS),Windows_NT)
    DETECTED_OS := Windows
    EXE_EXT := .exe
    RM_CMD := cmd /C rmdir /S /Q
    MKDIR_CMD := cmd /C mkdir
    CP_CMD := cmd /C copy
    MV_CMD := cmd /C move
    PATH_SEP := \\
    # Convert forward slashes to backslashes for Windows
    fixpath = $(subst /,\,$1)
else
    DETECTED_OS := $(shell uname -s)
    EXE_EXT :=
    RM_CMD := rm -rf
    MKDIR_CMD := mkdir -p
    CP_CMD := cp
    MV_CMD := mv
    PATH_SEP := /
    # Keep paths as-is for Unix
    fixpath = $1
endif

# Variables
DIR := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
BINARY_DIR := $(DIR)/bin
REACT_DIR := $(DIR)/src/IBDash
PUBLIC_DIR := $(DIR)/public

# Binary names
BINARY_DNS := ibpdns$(EXE_EXT)
BINARY_MONITOR := ibpmonitor$(EXE_EXT)
BINARY_COLLATOR := ibpcollator$(EXE_EXT)

# Build paths
BUILD_DNS := ./src/IBPDns
BUILD_MONITOR := ./src/IBPMonitor
BUILD_COLLATOR := ./src/IBPCollator

all: build

build: build-dns build-monitor build-collator dashboard-build

build-dns:
	$(GOBUILD) -o $(BINARY_DIR)/$(BINARY_DNS) -v $(BUILD_DNS)

build-monitor:
	$(GOBUILD) -o $(BINARY_DIR)/$(BINARY_MONITOR) -v $(BUILD_MONITOR)

build-collator:
	$(GOBUILD) -o $(BINARY_DIR)/$(BINARY_COLLATOR) -v $(BUILD_COLLATOR)

clean:
	$(GOCLEAN)
	-$(RM_CMD) $(call fixpath,$(BINARY_DIR))
	-$(RM_CMD) $(call fixpath,$(PUBLIC_DIR))

test:
	$(GOTEST) -v ./...

run-dns: build-dns
	cd $(DIR) && $(BINARY_DIR)/$(BINARY_DNS) -config=./config/ibpdns.json

run-monitor: build-monitor
	cd $(DIR) && $(BINARY_DIR)/$(BINARY_MONITOR) -config=./config/ibpmonitor.json

run-collator: build-collator
	cd $(DIR) && $(BINARY_DIR)/$(BINARY_COLLATOR) -config=./config/ibpcollator.json

# React Dashboard commands
dashboard-install:
	cd $(REACT_DIR) && npm install

dashboard-build:
	cd $(REACT_DIR) && npm run build
	-$(RM_CMD) $(call fixpath,$(PUBLIC_DIR))
ifeq ($(OS),Windows_NT)
	$(MV_CMD) $(call fixpath,$(REACT_DIR)/build) $(call fixpath,$(PUBLIC_DIR))
	$(MKDIR_CMD) $(call fixpath,$(PUBLIC_DIR)/static/imgs)
	$(CP_CMD) $(call fixpath,$(DIR)/assets/ibp.png) $(call fixpath,$(PUBLIC_DIR)/static/imgs/)
else
	$(MV_CMD) $(REACT_DIR)/build $(PUBLIC_DIR)
	$(MKDIR_CMD) $(PUBLIC_DIR)/static/imgs
	$(CP_CMD) $(DIR)/assets/ibp.png $(PUBLIC_DIR)/static/imgs/
endif

dashboard-dev:
ifeq ($(OS),Windows_NT)
	$(MKDIR_CMD) $(call fixpath,$(REACT_DIR)/public/static/imgs) 2>NUL || echo.
	$(CP_CMD) $(call fixpath,$(DIR)/assets/ibp.png) $(call fixpath,$(REACT_DIR)/public/static/imgs/)
else
	$(MKDIR_CMD) $(REACT_DIR)/public/static/imgs
	$(CP_CMD) $(DIR)/assets/ibp.png $(REACT_DIR)/public/static/imgs/
endif
	cd $(REACT_DIR) && npm start

# Combined commands
install: dashboard-install

.PHONY: all build build-dns build-monitor build-collator clean test run-dns run-monitor run-collator dashboard-install dashboard-build dashboard-dev install