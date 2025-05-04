# Makefile at your top-level project directory

# Default target: build all binaries
.PHONY: all
all: mgmt-matrixbot mgmt-discordbot mgmt-api powerdns-backend

# Ensure the bin/ directory exists
bin:
	mkdir -p bin

# mgmt-matrixbot
mgmt-matrixbot: bin
	go build -o bin\mgmt-matrixbot .\src\mgmt-matrixbox\mgmt-matrixbot.go

# mgmt-discordbot
mgmt-discordbot: bin
	go build -o bin\mgmt-discordbot .\src\mgmt-discordbot\mgmt-discordbot.go

# mgmt-api
mgmt-api: bin
	go build -o bin\mgmt-api .\src\mgmt-api\mgmt-api.go

# powerdns-backend
powerdns-backend: bin
	go build -o bin\powerdns-backend .\src\powerdns-backend\powerdns-backend.go

# Cleanup
.PHONY: clean
clean:
	rm -rf bin
