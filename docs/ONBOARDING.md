# Onboarding a new GeoDNS node

End-to-end checklist for adding a new IBP member's GeoDNS node.

## What you need before starting

1. A fresh Ubuntu 24.04 VM (4 vCPU, 16 GB RAM, 50 GB disk minimum) with:
   - public IPv4 (and ideally IPv6) reachable from anywhere
   - root SSH access via a key you control (cloud-init or manual install)
   - outbound HTTPS allowed
2. Admin access to set GitHub secrets on `ibp-network/ibp-geodns`.
3. (Optional but recommended) Maxmind GeoLite2 account — free signup at
   <https://www.maxmind.com/en/geolite2/signup>.

## One-time secret setup

Replace `<ORG>` with your organization name (`ROTKO`, `STAKEPLUS`, etc.).
**All secrets are per-organization** so different operators don't share
NATS or Maxmind credentials. Set these repo-level secrets:

| Secret | Contents |
|---|---|
| `ROOT_KEY_<ORG>` | Private SSH key that lets `root@your-node` log in (used **only** by the install workflow) |
| `DEPLOY_KEY_<ORG>` | Private SSH key the deploy workflow uses to ssh as `geodns@your-node` (generate a fresh one — do not reuse) |
| `MAXMIND_ACCOUNT_ID_<ORG>` | Numeric Maxmind account ID (`123456`) |
| `MAXMIND_LICENSE_KEY_<ORG>` | Maxmind GeoIP Update license key |
| `NATS_URL_<ORG>` | NATS client URL, e.g. `nats://127.0.0.1:4222` (if NATS runs locally on the node) or `nats://shared.host:4222` |
| `NATS_USER_<ORG>` | NATS user (typically matches the org name, e.g. `ROTKO`) |
| `NATS_PASS_<ORG>` | NATS password for that user |
| `MONITOR_URL_<ORG>` | Where IBPDns polls for health snapshots, e.g. `http://127.0.0.1:6101` if the local monitor is co-installed |

Generate a fresh deploy keypair and upload the private half:

```bash
ssh-keygen -t ed25519 -N '' -C 'geodns-deploy-<ORG>' -f geodns-deploy-<ORG>
gh secret set DEPLOY_KEY_<ORG> --repo ibp-network/ibp-geodns < geodns-deploy-<ORG>
# DO NOT commit/keep the private key anywhere else; the GH secret is canonical.
shred -u geodns-deploy-<ORG>
```

The corresponding public key is what the install workflow writes onto the
new node — you don't need to push it manually.

## Provision the node

Trigger the install workflow once per new node:

```bash
gh workflow run install.yaml --repo ibp-network/ibp-geodns \
  -f organization=<ORG> \
  -f server=<public-ip-or-hostname> \
  -f node_id=<ORG>-<SITE>-01 \
  -f run_deploy_after=true
gh run watch --repo ibp-network/ibp-geodns
```

What it does (`scripts/install-node.sh`) — Tom's canonical layout:

1. Installs OS packages (qemu-guest-agent, ufw, mysql-server, pdns-server,
   pdns-backend-remote)
2. Creates the `ibp` system user (or fixes its home/shell if it already
   exists), authorizes the deploy ssh key
3. Provisions `/opt/ibp-geodns/{bin,.staging}` (+ sibling dirs for monitor
   and collator) owned by `ibp`
4. Creates the local MySQL database/user; password stashed at
   `/root/.ibpdns_mysql_pass`
5. Drops `/etc/ibpdns/dns.json` (Maxmind+NATS values are blanks here and get
   filled in by deploy.yaml from secrets)
6. Installs the system-level `ibpdns.service` unit
7. Sudoers grant for `ibp` user to manage ibpdns / ibpmonitor / ibpcollator
   / pdns services
8. Configures PowerDNS as a remote backend pointing at IBPDns on
   `127.0.0.1:6100`, listening on both v4 and v6
9. Configures UFW to allow only 22/tcp + 53/udp,tcp inbound

The script **detects existing components and skips them** — if NATS is
already running (`/etc/ibpdns/nats.conf` present), it leaves that alone;
same for `monitor.json` and `collator.json`. Re-runs safely converge to
desired state.

Monitor and collator are separate binaries from sibling repos
(`ibp-network/ibp-geodns-monitor`, `ibp-network/ibp-geodns-collator`).
`scripts/install-monitor.sh` and `scripts/install-collator.sh` here
provision the dirs/unit/config for those services, and the deploy
workflows should live in their respective repos.

When `run_deploy_after=true`, the install workflow chains directly into the
deploy workflow once it succeeds. After that step you should see:

- `geodns@your-node:~$ systemctl --user is-active ibpdns` → `active`
- `dig @your-node-ip <some-zone-served-by-ibp>` returns answers

## Day-to-day deploy

Pushing a new build:

```bash
gh workflow run deploy.yaml --repo ibp-network/ibp-geodns \
  -f organization=<ORG> \
  -f server=<your-node> \
  -f version=main \
  -f restart_services=true
```

Each deploy:

1. Builds the linux/amd64 `IBPDns` binary with reproducible flags
2. rsyncs it to `/opt/geodns/.staging/<commit>/`, verifies sha256, promotes
3. Patches `/etc/ibpdns/ibpdns.json` with the latest secret values
4. `systemctl --user daemon-reload` + `restart ibpdns`
5. Verifies the unit becomes `active`; rolls back on failure

## Package updates

`scripts/update-node.sh` (driven by `update.yaml`) does an apt update +
upgrade on the node. The IBPDns binary itself isn't touched — it's deployed
by `deploy.yaml`, not by apt.

```bash
gh workflow run update.yaml --repo ibp-network/ibp-geodns \
  -f organization=<ORG> \
  -f server=<your-node> \
  -f auto_reboot=false   # set true if you want a reboot when /var/run/reboot-required appears
```

The script emits `REBOOT-REQUIRED yes|no` in the CI log so you can grep for
kernel updates.

## Rotating secrets

Just `gh secret set MAXMIND_LICENSE_KEY` (etc.) and trigger a deploy — the
patch step in the deploy workflow rewrites `ibpdns.json` from the current
secret values on every run.

## DNS delegation

Once the node is healthy and answering on port 53, the registrar's NS records
for `dotters.network` need updating to include the new node. Coordinate with
whoever owns the registrar account before flipping that.

## Troubleshooting

- `gh run view <id> --log-failed` shows the failing step's output.
- On the node: `journalctl --user -u ibpdns -n 50 --no-pager`
- Re-run the installer to fix drift: rerun `install.yaml` with the same args.
- If the deploy workflow can't ssh in, regenerate `DEPLOY_KEY_<ORG>` and run
  install again — the installer overwrites `geodns`'s `authorized_keys`.
