# IBP-GeoDNS

IBP GeoDNS package is made up of the following:
- Common Components: Generic functionality re-used in multiple components
- Power DNS API: RPC Monitor system that provides an api for remote-backend of powerdns
- Management API: Rest API that enables various functionality, commands, and requests 
- Discord Bot: Discord bot that enables various functionality, commands and requests
- Matrix Bot: Matrix bot that enables various functionality, commands and requests

Installation:
- Install golang 1.24.0 or better is installed
- Install automake
- run "make all" from the root directory

Installation will build the following binaries
- bin/dnsApi
- bin/mgmtApi
- bin/mgmtBotDiscord
- bin/mgmtBotMatrix

Example systemd scripts located are located at docs/systemd/
Example configuration file is located at docs/config-example.json

Install the configuration file in config/config.json and execute the binaries with the "-conf config/config.json" flag.

## Common Components

### Config

### Data

### Logging

### Maxmind

### Signal



## PowerDNS API

### API Handler

### RPC Monitor



## Management API

### API Handler

### Signal Communication



## Management Discord Bot

The Discord bot listens for simple management commands within a Discord channel.
It requires a bot token configured in `config.json` under the `Discord` section.

### Setup

1. Create a Discord application and bot at <https://discord.com/developers/applications>.
2. Copy the bot token and place it in your configuration:

   ```json
   "Discord": {
       "Token": "your_discord_bot_token"
   }
   ```

3. Build the bot using `make mgmtBotDiscord` and run it with:

   ```sh
   ./bin/mgmtBotDiscord -config config/config.json
   ```

When a user sends `status` in a channel the bot can read, it replies with `online`.

## Management Matrix Bot

The Matrix bot provides similar functionality using the Matrix protocol. It logs
in with credentials specified in the `Matrix` section of `config.json` and listens
for messages in the configured room.

### Setup

1. Ensure the `Matrix` settings in your configuration contain the homeserver URL,
   username, password, and the target room ID.
2. Build the bot with `make mgmtBotMatrix` and start it:

   ```sh
   ./bin/mgmtBotMatrix -config config/config.json
   ```

Sending `status` in the room will cause the bot to respond with `online`.
