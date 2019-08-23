#!/bin/bash
set -e
DESTDIR=${DESTDIR:-/usr/local}

DESTBIN=$DESTDIR/bin
rm -f $DESTBIN/ndnfw-dpdk $DESTBIN/ndnping-dpdk $DESTBIN/ndndpdk-*

DESTNODE=$DESTDIR/lib/node_modules/@usnistgov/ndn-dpdk
rm -rf $DESTNODE
