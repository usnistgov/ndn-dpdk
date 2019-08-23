#!/bin/bash
set -e
DESTDIR=${DESTDIR:-/usr/local}

DESTSBIN=$DESTDIR/sbin
rm -f $DESTSBIN/ndnfw-dpdk $DESTSBIN/ndnping-dpdk $DESTSBIN/ndndpdk-*

DESTBIN=$DESTDIR/bin
rm -f $DESTBIN/ndndpdk-*

DESTNODE=$DESTDIR/lib/node_modules/@usnistgov/ndn-dpdk
rm -rf $DESTNODE
