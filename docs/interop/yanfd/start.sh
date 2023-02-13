#!/bin/bash
set -euo pipefail

cp /usr/local/etc/ndn/yanfd.toml.sample /usr/local/etc/ndn/yanfd.toml

yanfd_put() {
  dasel put -f /usr/local/etc/ndn/yanfd.toml -t "$1" -s "$2" -v "$3"
}

yanfd_put bool .faces.tcp.enabled false
yanfd_put string .faces.unix.socket_path /run/ndn/yanfd.sock
yanfd_put bool .faces.websocket.enabled false

exec yanfd
