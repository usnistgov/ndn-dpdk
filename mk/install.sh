#!/bin/bash
set -e
set -o pipefail
DESTDIR=${DESTDIR:-/usr/local}

DESTSBIN=$DESTDIR/sbin
install -d -m0755 $DESTSBIN
install -m0744 build/bin/ndnfw-dpdk $DESTSBIN/
install -m0744 build/bin/ndnping-dpdk $DESTSBIN/

DESTBIN=$DESTDIR/bin
install -d -m0755 $DESTBIN
install -m0755 build/bin/ndndpdk-hrlog2histogram $DESTBIN/
install -m0755 cmd/mgmtclient/mgmtcmd.sh $DESTBIN/ndndpdk-mgmtcmd

npm install -C /usr/local -g ./build/ndn-dpdk.tgz
