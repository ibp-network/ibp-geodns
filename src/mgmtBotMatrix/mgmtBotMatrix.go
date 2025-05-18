package main

import (
	"flag"
	"os"

	cfg "ibp-geodns/src/common/config"
	log "ibp-geodns/src/common/logging"
	"ibp-geodns/src/mgmtBotMatrix/matrix"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
)

var version = "0.1.0"

func main() {
	log.SetLogLevel(log.Info)
	log.Log(log.Info, "Mgmt Matrix Bot v%s starting...", version)

	cfgFile := flag.String("config", "config.json", "Path to the configuration file")
	flag.Parse()

	if _, err := os.Stat(*cfgFile); os.IsNotExist(err) {
		log.Log(log.Fatal, "Configuration file not found: %s", *cfgFile)
		os.Exit(1)
	}

	cfg.Init(*cfgFile)
	conf := cfg.GetConfig()
	mconf := conf.Local.Matrix

	bot, err := matrix.NewMatrixBot(mconf.HomeServerURL, mconf.Username, mconf.Password, mconf.RoomID)
	if err != nil {
		log.Log(log.Fatal, "Failed to initialize Matrix bot: %v", err)
		os.Exit(1)
	}

	syncer := bot.Client.Syncer.(*mautrix.DefaultSyncer)
	// Register a basic event handler that does nothing.
	syncer.OnEventType(event.EventMessage, mautrix.EventHandler(func(mautrix.EventSource, *event.Event) {
		// no-op handler
	}))

	log.Log(log.Info, "Matrix bot is now running")
	if err := bot.Client.Sync(); err != nil {
		log.Log(log.Fatal, "Sync failed: %v", err)
	}
}
