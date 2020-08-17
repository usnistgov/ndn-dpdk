#!/bin/bash
set -e
set -o pipefail
DESTDIR=${DESTDIR:-/usr/local}

DESTBPF=$DESTDIR/lib/bpf
rm -rf $DESTBPF/ndndpdk-strategy-*.o

DESTSBIN=$DESTDIR/sbin
rm -f $DESTSBIN/ndnfw-dpdk $DESTSBIN/ndnping-dpdk $DESTSBIN/ndndpdk-*

DESTBIN=$DESTDIR/bin
rm -f $DESTBIN/ndndpdk-*

DESTNODE=$DESTDIR/lib/node_modules/@usnistgov/ndn-dpdk
rm -rf $DESTNODE
