# IBP-GeoDNS

IBP GeoDNS package is made up of the following:
- Common Components: Generic functionality re-used in multiple components
- Power DNS API: RPC Monitor system that provides an api for remote-backend of powerdns
- Management API: Rest API that enables various fucntionality, commands, and requests 
- Discord Bot: Discord bot that enables various functionality, commands and requests
- Matrix Bot: Matrix bot that enables vbarious functionality, commands and requests

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

### Discord Handler

### Signal Communication



## Management Matrix Bot

### Matrix Handler

### Signal Communications
