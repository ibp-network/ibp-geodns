## README.md
```markdown
# IBP GeoDNS

Distributed DNS backend for PowerDNS with health monitoring and geo-location routing.

## Server Setup

```bash
# Create geodns user
sudo useradd -r -s /bin/bash -m -d /opt/geodns geodns
sudo loginctl enable-linger geodns

# Add SSH key for deployments
sudo -u geodns mkdir -p /opt/geodns/.ssh
sudo -u geodns tee /opt/geodns/.ssh/authorized_keys < your-deploy-key.pub
```

## Configuration

Create config file:
```bash
sudo -u geodns mkdir -p /opt/geodns/config
sudo -u geodns vi /opt/geodns/config/config.json # edit/create cfg
```

## Services

Create all systemd services:
```bash
sudo -u geodns mkdir -p /opt/geodns/.config/systemd/user

# IBPDns service
sudo -u geodns tee /opt/geodns/.config/systemd/user/ibpdns.service <<EOF
[Unit]
Description=IBP DNS Backend

[Service]
Type=simple
WorkingDirectory=/opt/geodns
ExecStart=/opt/geodns/bin/IBPDns -config /opt/geodns/config/config.json
Restart=always

[Install]
WantedBy=default.target
EOF

# IBPMonitor service
sudo -u geodns tee /opt/geodns/.config/systemd/user/ibpmonitor.service <<EOF
[Unit]
Description=IBP Monitor

[Service]
Type=simple
WorkingDirectory=/opt/geodns
ExecStart=/opt/geodns/bin/IBPMonitor -config /opt/geodns/config/config.json
Restart=always

[Install]
WantedBy=default.target
EOF

# IBPCollator service
sudo -u geodns tee /opt/geodns/.config/systemd/user/ibpcollator.service <<EOF
[Unit]
Description=IBP Collator

[Service]
Type=simple
WorkingDirectory=/opt/geodns
ExecStart=/opt/geodns/bin/IBPCollator -config /opt/geodns/config/config.json
Restart=always

[Install]
WantedBy=default.target
EOF

# Enable services as needed(with geodns user or root/sudo)
sudo -u geodns systemctl --user daemon-reload
sudo -u geodns systemctl --user enable --now ibpdns
sudo -u geodns systemctl --user enable --now ibpmonitor
sudo -u geodns systemctl --user enable --now ibpcollator
```

## Deploy

### GitHub Actions

1. Add ssh-key at [github cfg](https://github.com/ibp-network/config/settings/secrets/actions) DEPLOY_KEY_<YOUR_ORG>
2. Add <YOUR_ORG> in flow deploy options
3. Go to [Actions](https://github.com/ibp-network/ibp-geodns/actions/workflows/deploy.yml)
4. Run workflow → Select organization, server, version
5. Builds and deploys to selected server location as geodns user

### Manual deploy

```bash
# Build
go mod tidy
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o IBPDns src/IBPDns/IBPDns.go
# and so on

# Deploy
scp IBPDns geodns@server:/opt/geodns/v1.0/
ssh geodns@server "cd /opt/geodns && ln -sfn v1.0/IBPDns bin/IBPDns"
```

## PowerDNS

```
launch=remote
remote-connection-string=http:url=http://127.0.0.1:6100/dns
```
