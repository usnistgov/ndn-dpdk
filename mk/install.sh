#!/bin/bash
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"/..
source mk/install-dirs.sh

install -d -m0755 "$DESTBPF"
install -m0644 build/lib/bpf/*.o "$DESTBPF/"

install -d -m0755 "$DESTBIN"
install -m0755 build/bin/ndndpdk-ctrl "$DESTBIN/"
install -m0755 build/bin/ndndpdk-godemo "$DESTBIN/"
install -m0755 build/bin/ndndpdk-hrlog2histogram "$DESTBIN/"
install -m0755 build/bin/ndndpdk-jrproxy "$DESTBIN/"
install -m0755 build/bin/ndndpdk-svc "$DESTBIN/"

install -d -m0755 "$DESTSHARE"
install -m0644 build/share/ndn-dpdk/* "$DESTSHARE/"

install -d -m0755 "$DESTBASHCOMP"
for CMD in ndndpdk-ctrl ndndpdk-godemo; do
  install -m0644 build/share/bash_autocomplete "$DESTBASHCOMP/$CMD"
done

install -d -m0755 "$DESTSYSTEMD"
install -m0644 docs/ndndpdk-*.service "$DESTSYSTEMD/"
if command -v systemctl >/dev/null; then
  systemctl daemon-reload
fi
