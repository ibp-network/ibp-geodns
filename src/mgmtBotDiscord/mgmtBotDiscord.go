package main

import (
	"flag"
	"os"

	"github.com/bwmarrin/discordgo"
	cfg "ibp-geodns/src/common/config"
	log "ibp-geodns/src/common/logging"
)

var version = "0.1.0"

func main() {
	log.SetLogLevel(log.Info)
	log.Log(log.Info, "Mgmt Discord Bot v%s starting...", version)

	cfgFile := flag.String("config", "config.json", "Path to the configuration file")
	flag.Parse()

	if _, err := os.Stat(*cfgFile); os.IsNotExist(err) {
		log.Log(log.Fatal, "Configuration file not found: %s", *cfgFile)
		os.Exit(1)
	}

	cfg.Init(*cfgFile)
	conf := cfg.GetConfig()

	token := conf.Local.Discord.Token
	if token == "" {
		log.Log(log.Fatal, "Discord token not configured")
		os.Exit(1)
	}

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Log(log.Fatal, "Error creating Discord session: %v", err)
		os.Exit(1)
	}

	dg.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.ID == s.State.User.ID {
			return
		}
		if m.Content == "status" {
			_, err := s.ChannelMessageSend(m.ChannelID, "online")
			if err != nil {
				log.Log(log.Error, "failed to send message: %v", err)
			}
		}
	})

	err = dg.Open()
	if err != nil {
		log.Log(log.Fatal, "Error opening Discord session: %v", err)
		os.Exit(1)
	}

	log.Log(log.Info, "Discord bot is now running")
	select {}
}
