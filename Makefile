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

# Build paths
BUILD_DNS=./src/IBPDns
BUILD_MONITOR=./src/IBPMonitor
BUILD_COLLATOR=./src/IBPCollator

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

# React Dashboard commands
dashboard-install:
	npm install

dashboard-build:
	mkdir ./public/static/imgs
	cp assets/ibp.png ./public/static/imgs/
	cd ./src/IBDash/
	npm run build

dashboard-dev:
	mkdir ./public/static/imgs
	cp assets/ibp.png ./public/static/imgs/
	cd ./src/IBDash/
	npm start

# Combined commands
install: dashboard-install

build-all: build dashboard-build

.PHONY: all build build-dns build-monitor build-collator clean test run-dns run-monitor run-collator dashboard-install dashboard-build dashboard-dev install build-all