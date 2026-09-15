#!/bin/sh
# Run from an uploaded release directory containing obsidianchat and install.sh.
set -eu
release=${1:?release required}
hash=${2:?SHA256 required}
case "$release" in ''|*[!0-9]*) exit 1 ;; esac
case "$hash" in ''|*[!0-9a-f]*) exit 1 ;; esac
test "${#hash}" -eq 64
cd "$(dirname "$0")"
printf '%s  obsidianchat\n' "$hash" | sha256sum -c -
test -d /data/obsidianchat
install -d -m 0700 /opt/obsidianchat/backups
backup="/opt/obsidianchat/backups/before-$release.tar.gz"
test ! -e "$backup"
# Stop only long enough to capture SQLite together with its WAL and uploaded files.
trap 'systemctl start obsidianchat || true' EXIT
systemctl stop obsidianchat
tar -C /data -czf "$backup" obsidianchat
chmod 0600 "$backup"
tar -tzf "$backup" >/dev/null
systemctl start obsidianchat
trap - EXIT
sh ./install.sh "$release" "$hash"
printf '\nBackup: %s\n' "$backup"
readlink -f /opt/obsidianchat/current
