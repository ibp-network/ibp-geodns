package main

import (
	"flag"
	"os"
	"time"

	"common/data"
	l "common/logging"
	"common/networking"

	"common/config"
)

var version = "0.2.0"

func main() {
	// Initialize the logging level
	l.SetLogLevel(l.Debug)
	l.Log(l.Info, "IBP-GeoDNS v%s starting...", version)

	// Define a command-line flag for the config file path
	cfgFile := flag.String("config", "config.json", "Path to the configuration file")
	flag.Parse()

	// Check if the provided config file exists
	if _, err := os.Stat(*cfgFile); os.IsNotExist(err) {
		l.Log(l.Fatal, "Configuration file not found: %s", *cfgFile)
		os.Exit(1)
	}

	// Initialize components
	config.Init(*cfgFile)

	// Start data package
	data.Init()

	// Sleep to load the caches load
	time.Sleep(1 * time.Second)

	// Init networking and monitoring
	networking.Init()

	// Infinite loop to keep things operational
	loop()
}

func loop() {
	for {
		time.Sleep(60 * time.Second)
	}
}
