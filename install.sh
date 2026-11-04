#!/bin/sh
set -eu

project_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
bin_dir="${HOME}/bin"
mkdir -p "$bin_dir"
go build -o "$bin_dir/shishimon-server" "$project_dir"
cat > "$bin_dir/bro-start" <<'SCRIPT'
#!/bin/sh
set -eu
bin_dir="${HOME}/bin"
pid_file="${HOME}/.shishimon.pid"
log_file="${HOME}/.shishimon.log"

if [ -f "$pid_file" ]; then
  pid=$(cat "$pid_file")
  case "$pid" in
    ''|*[!0-9]*) ;;
    *) if kill -0 "$pid" 2>/dev/null; then
         echo "Shishimon is already running (PID $pid). Use bro-stop first."
         exit 1
       fi ;;
  esac
  rm -f "$pid_file"
fi

nohup "$bin_dir/shishimon-server" > "$log_file" 2>&1 &
pid=$!
echo "$pid" > "$pid_file"
i=0
while [ "$i" -lt 10 ] && ! grep -q "URL:" "$log_file" 2>/dev/null; do
  if ! kill -0 "$pid" 2>/dev/null; then
    break
  fi
  sleep 1
  i=$((i + 1))
done
if ! kill -0 "$pid" 2>/dev/null || ! grep -q "URL:" "$log_file" 2>/dev/null; then
  cat "$log_file"
  rm -f "$pid_file"
  exit 1
fi
cat "$log_file"
echo "Shishimon is running in the background (PID $pid)."
echo "Use bro-stop to stop it."
SCRIPT
chmod +x "$bin_dir/bro-start"
cat > "$bin_dir/bro-stop" <<'SCRIPT'
#!/bin/sh
set -eu
pid_file="${HOME}/.shishimon.pid"
if [ ! -f "$pid_file" ]; then
  echo "Shishimon is not running."
  exit 0
fi
pid=$(cat "$pid_file")
case "$pid" in
  ''|*[!0-9]*) echo "Invalid Shishimon PID file."; exit 1 ;;
esac
if kill -0 "$pid" 2>/dev/null; then
  kill "$pid"
  echo "Shishimon stopped (PID $pid)."
else
  echo "Shishimon was not running."
fi
rm -f "$pid_file"
SCRIPT
chmod +x "$bin_dir/bro-stop"
echo "Installed shishimon-server, bro-start, and bro-stop to $bin_dir"
echo "If ~/bin is not on PATH, add: export PATH=\"\$HOME/bin:\$PATH\""
