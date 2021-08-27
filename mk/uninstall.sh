#!/bin/bash
set -eo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"/..
source mk/install-dirs.sh

rm -f $DESTBPF/ndndpdk-strategy-*.o

rm -f $DESTBIN/ndndpdk-*

rm -rf $DESTSHARE

rm -f $DESTSYSTEMD/ndndpdk-*.service
if which systemctl >/dev/null; then
  systemctl daemon-reload
fi
