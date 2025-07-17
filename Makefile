# Variables
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BINARY_DIR=./bin
REACT_DIR=./src/IBDash

# Binary names
BINARY_DNS=ibpdns
BINARY_MONITOR=ibpmonitor
BINARY_COLLATOR=ibpcollator
BINARY_MGMT=ibpmgmt
BINARY_BOT_DISCORD=ibpbotdiscord
BINARY_BOT_MATRIX=ibpbotmatrix

# Build paths
BUILD_DNS=./src/IBPDns
BUILD_MONITOR=./src/IBPMonitor
BUILD_COLLATOR=./src/IBPCollator
BUILD_MGMT=./src/IBPMgmt
BUILD_BOT_DISCORD=./src/IBPBotDiscord
BUILD_BOT_MATRIX=./src/IBPBotMatrix

all: build

build: build-dns build-monitor build-collator build-mgmt build-bot-discord build-bot-matrix

build-dns:
	$(GOBUILD) -o $(BINARY_DIR)/$(BINARY_DNS) -v $(BUILD_DNS)

build-monitor:
	$(GOBUILD) -o $(BINARY_DIR)/$(BINARY_MONITOR) -v $(BUILD_MONITOR)

build-collator:
	$(GOBUILD) -o $(BINARY_DIR)/$(BINARY_COLLATOR) -v $(BUILD_COLLATOR)

build-mgmt:
	$(GOBUILD) -o $(BINARY_DIR)/$(BINARY_MGMT) -v $(BUILD_MGMT)

build-bot-discord:
	$(GOBUILD) -o $(BINARY_DIR)/$(BINARY_BOT_DISCORD) -v $(BUILD_BOT_DISCORD)

build-bot-matrix:
	$(GOBUILD) -o $(BINARY_DIR)/$(BINARY_BOT_MATRIX) -v $(BUILD_BOT_MATRIX)

clean:
	$(GOCLEAN)
	rm -rf $(BINARY_DIR)

test:
	$(GOTEST) -v ./...

run-dns:
	$(GOBUILD) -o $(BINARY_DIR)/$(BINARY_DNS) -v $(BUILD_DNS)
	$(BINARY_DIR)/$(BINARY_DNS) -config=./config/ibpdns.json

run-monitor:
	$(GOBUILD) -o $(BINARY_DIR)/$(BINARY_MONITOR) -v $(BUILD_MONITOR)
	$(BINARY_DIR)/$(BINARY_MONITOR) -config=./config/ibpmonitor.json

run-collator:
	$(GOBUILD) -o $(BINARY_DIR)/$(BINARY_COLLATOR) -v $(BUILD_COLLATOR)
	$(BINARY_DIR)/$(BINARY_COLLATOR) -config=./config/ibpcollator.json

run-mgmt:
	$(GOBUILD) -o $(BINARY_DIR)/$(BINARY_MGMT) -v $(BUILD_MGMT)
	$(BINARY_DIR)/$(BINARY_MGMT) -config=./config/ibpmgmt.json

run-bot-discord:
	$(GOBUILD) -o $(BINARY_DIR)/$(BINARY_BOT_DISCORD) -v $(BUILD_BOT_DISCORD)
	$(BINARY_DIR)/$(BINARY_BOT_DISCORD) -config=./config/ibpbotdiscord.json

run-bot-matrix:
	$(GOBUILD) -o $(BINARY_DIR)/$(BINARY_BOT_MATRIX) -v $(BUILD_BOT_MATRIX)
	$(BINARY_DIR)/$(BINARY_BOT_MATRIX) -config=./config/ibpbotmatrix.json

# React Dashboard commands
dashboard-install:
	cd $(REACT_DIR) && npm install

dashboard-build:
	mkdir -p $(REACT_DIR)/public/static/imgs
	cp assets/ibp.png $(REACT_DIR)/public/static/imgs/
	cd $(REACT_DIR) && npm run build

dashboard-dev:
	mkdir -p $(REACT_DIR)/public/static/imgs
	cp assets/ibp.png $(REACT_DIR)/public/static/imgs/
	cd $(REACT_DIR) && npm start

# Combined commands
install: dashboard-install

build-all: build dashboard-build

.PHONY: all build build-dns build-monitor build-collator build-mgmt build-bot-discord build-bot-matrix clean test run-dns run-monitor run-collator run-mgmt run-bot-discord run-bot-matrix dashboard-install dashboard-build dashboard-dev install build-all
