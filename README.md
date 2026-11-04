# Shishimon

Shishimon is a small local Mac dashboard written in Go. It shows live CPU, memory, memory pressure, swap, battery, uptime, and top processes in a phone-friendly web page.

## Install

From this directory:

```sh
./install.sh
export PATH="$HOME/bin:$PATH"
bro-start
```

`bro-start` starts Shishimon in the background, prints a QR code and a local URL, and returns control to the terminal. Scan the URL from a phone connected to the same Wi-Fi network. Stop it with:

```sh
bro-stop
```

The access code is a five-digit numeric token. The dashboard refreshes every two seconds over a server-sent event stream.

The server listens on port `8080` by default. Change it with:

```sh
SHISHIMON_PORT=9090 bro-start
```

Every launch creates a new access token. The memory-used value is calculated from active and wired pages plus compressed memory; free, inactive, speculative, and purgeable pages are not counted as actively consumed. The service is intended for a trusted local network and does not expose itself to the public internet.
