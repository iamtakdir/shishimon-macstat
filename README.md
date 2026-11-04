# Shishimon

### A futuristic, real-time Mac system monitor for your phone

[![Go](https://img.shields.io/badge/Go-1.27+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Platform](https://img.shields.io/badge/platform-macOS-000000?logo=apple&logoColor=white)](https://www.apple.com/macos/)
[![License](https://img.shields.io/badge/license-not--specified-lightgrey)](https://github.com/iamtakdir/shishimon-macstat)
[![GitHub stars](https://img.shields.io/github/stars/iamtakdir/shishimon-macstat?style=flat&logo=github)](https://github.com/iamtakdir/shishimon-macstat/stargazers)

Shishimon is a tiny Go web service that turns your Mac into a live system-monitoring station. Run one command in Terminal, scan the QR code with your phone, and watch your Mac's telemetry in real time from anywhere on the same local network.

```text
┌─────────────────────────────────────────────────────────────┐
│  [◈] SHISHIMON                              ● LIVE           │
│  LOCAL SYSTEM TELEMETRY // v1.0                             │
├─────────────────────────────────────────────────────────────┤
│  CPU LOAD       24.0 %       MEMORY USED      62.3 %         │
│  PRESSURE       NORMAL       SWAP             35.5 MB        │
│  BATTERY        100 %        UPTIME           19h 22m        │
├─────────────────────────────────────────────────────────────┤
│  SHISHIMON PROCESS // SELF-MONITOR                         │
│  CPU 0.02 %     RAM 6.5 MB     PID 18376                   │
└─────────────────────────────────────────────────────────────┘
```

## Features

- **Live dashboard** — CPU, memory, pressure, swap, battery, uptime, and top processes.
- **Phone-first UI** — responsive terminal-inspired design with no horizontal scrolling.
- **One shared collector** — metrics are collected once every two seconds and shared with all connected clients.
- **Self-monitoring** — see Shishimon's own CPU, resident memory, and PID.
- **QR-ready startup** — prints a scannable QR code and local URL in Terminal.
- **Simple lifecycle** — `bro-start` runs in the background; `bro-stop` shuts it down.
- **Local-network focused** — uses a fresh access code per launch and is intended to be reachable only from your trusted LAN.
- **No database** — stateless, lightweight, and easy to remove.

## Requirements

- macOS
- Go 1.27 or newer
- A phone and Mac connected to the same Wi-Fi or local network

## Install

Clone the repository and run the installer:

```sh
git clone git@github.com:iamtakdir/shishimon-macstat.git
cd shishimon-macstat
./install.sh
```

The installer builds the server and installs these commands into `~/bin`:

```text
~/bin/shishimon-server
~/bin/bro-start
~/bin/bro-stop
```

If `~/bin` is not already on your `PATH`:

```sh
export PATH="$HOME/bin:$PATH"
```

To keep that change after restarting Terminal:

```sh
echo 'export PATH="$HOME/bin:$PATH"' >> ~/.zshrc
```

## Start the dashboard

```sh
bro-start
```

Shishimon starts in the background and prints a URL similar to:

```text
URL: http://192.168.1.42:8080/?token=48291
```

It also prints a QR code. Scan it with your phone's camera while the phone is connected to the same network.

Stop the service at any time:

```sh
bro-stop
```

Runtime details:

| Item | Location |
|---|---|
| PID file | `~/.shishimon.pid` |
| Startup log | `~/.shishimon.log` |
| Default port | `8080` |
| Update interval | 2 seconds |
| Access code | Fresh five-digit numeric code per launch |

## Configuration

Change the listening port with `SHISHIMON_PORT`:

```sh
SHISHIMON_PORT=9090 bro-start
```

If another Shishimon process is already running, stop it first:

```sh
bro-stop
SHISHIMON_PORT=9090 bro-start
```

## What is measured?

| Metric | Source / behavior |
|---|---|
| CPU | macOS `top` user + system utilization |
| Memory total | `hw.memsize` |
| Memory used | Active pages + wired pages + compressed memory |
| Memory pressure | `vm.memory_pressure` |
| Swap | `vm.swapusage` |
| Battery | `pmset -g batt` |
| Uptime | `kern.boottime` |
| Processes | `ps`, sorted by CPU |
| Shishimon overhead | Shishimon's own CPU and resident memory |

Memory intentionally does not count free, inactive, speculative, or purgeable pages as actively used. This avoids the misleading 100% reading produced by macOS's rounded `top` output.

## Development

Run directly without installing:

```sh
go run .
```

Run the quality checks:

```sh
go test ./...
go test -race ./...
go vet ./...
go build ./...
```

The web dashboard is embedded into the Go binary from `web/index.html`, so the installed server does not need the source directory at runtime.

## Security notes

Shishimon is designed for a trusted local network, not the public internet.

- The access code is regenerated every time the service starts.
- The code is intentionally short and numeric for convenient phone entry.
- Do not forward port `8080` from your router.
- Do not expose the service on an untrusted public or shared network.
- The dashboard can reveal process names and machine resource usage.

## Troubleshooting

### The phone cannot connect

Confirm both devices are on the same network, then try the printed URL manually. Some guest Wi-Fi networks block device-to-device traffic.

### The port is already in use

Use another port:

```sh
bro-stop
SHISHIMON_PORT=9090 bro-start
```

### The startup log is needed

```sh
cat ~/.shishimon.log
```

### The command is not found

```sh
export PATH="$HOME/bin:$PATH"
```

## License

No license has been selected for this project yet. All rights are reserved unless the repository owner adds a license.

---

Built for quick glances, local networks, and terminal people.
