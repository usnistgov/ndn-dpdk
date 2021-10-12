#!/bin/bash
set -eo pipefail

cp /usr/local/etc/ndn/yanfd.toml.sample /usr/local/etc/ndn/yanfd.toml

yanfd_put() {
  dasel put "$1" -f /usr/local/etc/ndn/yanfd.toml "$2" "$3"
}

yanfd_put bool .faces.tcp.enabled false
yanfd_put string .faces.unix.socket_path /run/ndn/yanfd.sock
yanfd_put bool .faces.websocket.enabled false

exec yanfd
