#!/bin/bash
set -e
set -o pipefail
DESTDIR=${DESTDIR:-/usr/local}

DESTBPF=$DESTDIR/lib/bpf
rm -rf $DESTBPF/ndndpdk-strategy-*.o

DESTSBIN=$DESTDIR/sbin
DESTBIN=$DESTDIR/bin
rm -f $DESTSBIN/ndndpdk-* $DESTBIN/ndndpdk-*

DESTSHARE=$DESTDIR/share/ndn-dpdk
rm -rf $DESTSHARE
