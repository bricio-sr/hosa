#!/usr/bin/env bash
set -euo pipefail

if [[ "${EUID}" -ne 0 ]]; then
  echo "Run as root (sudo)." >&2
  exit 1
fi

install -d -m 0755 /etc/hosa
install -d -m 0755 /etc/default
install -d -m 0755 /etc/systemd/system

if [[ ! -f /etc/hosa/hosa.toml ]]; then
  install -m 0644 etc/hosa/hosa.toml /etc/hosa/hosa.toml
fi

cat >/etc/default/hosa-agent <<'EOF'
# Extra CLI flags for hosa_agent.
# Example: HOSA_EXTRA_FLAGS="--min-samples 60 --threshold-vigilance 4.0"
HOSA_EXTRA_FLAGS=""
EOF

install -m 0644 etc/systemd/hosa-agent.service /etc/systemd/system/hosa-agent.service

systemctl daemon-reload
systemctl enable --now hosa-agent.service
systemctl restart hosa-agent.service
systemctl --no-pager --full status hosa-agent.service
