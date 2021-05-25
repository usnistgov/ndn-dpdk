#!/bin/bash
set -eo pipefail
YANFD_CS_CAP=${YANFD_CS_CAP:-65536}

dasel select -f /usr/local/etc/ndn/yanfd.toml.sample -r toml -w json |
jq -c --arg csCap $YANFD_CS_CAP '.
  | setpath(["faces","unix","socket_path"]; "/run/ndn/yanfd.sock")
  | setpath(["tables","content_store","capacity"]; $csCap | tonumber)
' | dasel select -r json -w toml > /usr/local/etc/ndn/yanfd.toml

exec yanfd
