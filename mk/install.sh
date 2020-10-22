#!/bin/bash
set -e
set -o pipefail
DESTDIR=${DESTDIR:-/usr/local}

DESTBPF=$DESTDIR/lib/bpf
install -d -m0755 $DESTBPF
install -m0644 build/lib/bpf/*.o $DESTBPF/

DESTSBIN=$DESTDIR/sbin
install -d -m0755 $DESTSBIN
install -m0744 build/bin/ndndpdk-godemo $DESTSBIN/
install -m0744 build/bin/ndndpdk-svc $DESTSBIN/

DESTBIN=$DESTDIR/bin
install -d -m0755 $DESTBIN
install -m0755 build/bin/ndndpdk-ctrl $DESTBIN/
install -m0755 build/bin/ndndpdk-hrlog2histogram $DESTBIN/
install -m0755 cmd/mgmtclient/mgmtcmd.sh $DESTBIN/ndndpdk-mgmtcmd

DESTSHARE=$DESTDIR/share/ndn-dpdk
install -d -m0755 $DESTSHARE
install -m0644 build/share/ndn-dpdk/* $DESTSHARE/
