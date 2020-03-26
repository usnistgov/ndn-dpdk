#!/bin/bash
set -e
DESTDIR=${DESTDIR:-/usr/local}

DESTSBIN=$DESTDIR/sbin
install -d -m0755 $DESTSBIN
install -m0744 ../../bin/ndnfw-dpdk $DESTSBIN
install -m0744 ../../bin/ndnping-dpdk $DESTSBIN

DESTBIN=$DESTDIR/bin
install -d -m0755 $DESTBIN
install -m0755 ../../bin/ndndpdk-hrlog2histogram $DESTBIN
install -m0755 cmd/mgmtclient/mgmtcmd.sh $DESTBIN/ndndpdk-mgmtcmd

NPMTARBALL=$(npm pack -s .)
npm install -C /usr/local -g $NPMTARBALL
rm $NPMTARBALL
