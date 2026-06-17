# Deploying the relay on Oracle Cloud (Always Free)

The relay needs to stay always-on holding persistent WebSocket connections, so it
runs as a long-lived container on an Oracle **Always Free** VM. [Caddy](https://caddyserver.com)
sits in front to terminate TLS (browsers require `wss://` from an `https://` page)
and reverse-proxies to the Go relay. Caddy gets and renews a Let's Encrypt cert
automatically — you just point a domain at the VM.

```
browser ──https/wss──> Caddy :443 ──http/ws──> relay :8080
```

## 1. Create the VM

1. Sign in at <https://cloud.oracle.com> → **Compute → Instances → Create instance**.
2. **Shape:** pick an **Always Free–eligible** shape:
   - `VM.Standard.E2.1.Micro` (AMD, 1 OCPU / 1 GB) — plenty for this relay and
     almost always available. **Recommended.**
   - `VM.Standard.A1.Flex` (Ampere ARM, up to 4 OCPU / 24 GB) — more headroom, but
     free ARM capacity is often exhausted ("Out of host capacity"). Try the AMD
     micro first if ARM won't provision.
   - The build runs *inside* the container against the VM's native arch, so ARM vs
     x86 needs no change on your side.
3. **Image:** Ubuntu 24.04 (or 22.04).
4. **SSH keys:** upload your public key (or let Oracle generate one and download it).
5. Create, then note the instance's **public IPv4 address**.

## 2. Point your domain at the VM

Add a DNS **A record** in the `maos.dev` zone (this becomes `RELAY_DOMAIN`):

| Type | Name   | Value            |
|------|--------|------------------|
| A    | `peek` | `157.151.195.75` |

The `peek` name in the `maos.dev` zone resolves to `peek.maos.dev`.

> **Note — ephemeral IP.** `157.151.195.75` is an *ephemeral* public IP. It
> survives reboots and stop/start, and is only released if the instance is
> **terminated**. If you ever recreate the instance you'll get a new IP and must
> update this record. To make it permanent, convert it to a *reserved* public IP
> in the OCI console later.

Wait until `nslookup peek.maos.dev` resolves to the VM before continuing —
Let's Encrypt will fail if DNS hasn't propagated.

## 3. Open ports 80 and 443 — **in both firewalls**

Oracle has *two* layers and the #1 cause of "it just hangs" is forgetting the
second one.

**(a) Cloud firewall — VCN ingress rules**
Instance details → its **Subnet** → **Security List** (or NSG) → **Add Ingress Rules**:

| Source CIDR | Protocol | Destination port |
|-------------|----------|------------------|
| `0.0.0.0/0` | TCP      | 80               |
| `0.0.0.0/0` | TCP      | 443              |

**(b) Host firewall — Ubuntu ships with restrictive iptables**
SSH in (`ssh ubuntu@<public IP>`) and run:

```bash
sudo iptables -I INPUT 6 -m state --state NEW -p tcp --dport 80 -j ACCEPT
sudo iptables -I INPUT 6 -m state --state NEW -p tcp --dport 443 -j ACCEPT
sudo netfilter-persistent save
```

## 4. Install Docker

```bash
sudo apt-get update
sudo apt-get install -y ca-certificates curl git
curl -fsSL https://get.docker.com | sudo sh
sudo usermod -aG docker $USER
# log out and back in so the group change takes effect
```

## 5. Deploy

```bash
git clone <your-repo-url> peek
cd peek
cp .env.example .env
nano .env          # set RELAY_DOMAIN and ACME_EMAIL

docker compose up -d --build
```

Caddy requests the TLS cert on first start (a few seconds). Watch it:

```bash
docker compose logs -f caddy
```

## 6. Verify

```bash
curl https://peek.maos.dev/healthz      # -> 200, valid cert
```

Open `https://peek.maos.dev/watch?token=demo` in a browser — it should load
the viewer over a trusted cert (no warning). The agent connects to
`wss://peek.maos.dev/agent?token=<token>` and viewers to
`wss://peek.maos.dev/ws?token=<token>`.

## Operating notes

- **Updates:** `git pull && docker compose up -d --build`
- **Restart / boot:** `restart: unless-stopped` brings both containers back after a
  reboot. Docker's service is enabled on boot by default.
- **Certs persist** in the `caddy_data` volume, so restarts don't re-hit Let's
  Encrypt rate limits.
- **Cost:** Always Free shapes don't expire as long as you stay within free limits.
  Oracle reclaims *idle* Ampere instances on some accounts — the always-on relay
  traffic keeps it active. The AMD micro shape is not subject to that reclamation.
