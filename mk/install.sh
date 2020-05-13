#!/bin/bash
set -e
DESTDIR=${DESTDIR:-/usr/local}

go_install() {
  local PERM=$1
  local DEST=$2
  local PROGRAM=$3
  for SRCDIR in $(go env GOPATH | tr ':' ' '); do
    local SRC=$SRCDIR/bin/$PROGRAM
    if [[ -f $SRC ]]; then
      install -m$PERM $SRC $DEST
      return 0
    fi
  done
  echo 'Go program ' $PROGRAM 'not found. Did you run `make`?' >/dev/stderr
  return 1
}

DESTSBIN=$DESTDIR/sbin
install -d -m0755 $DESTSBIN
go_install 0744 $DESTSBIN ndnfw-dpdk
go_install 0744 $DESTSBIN ndnping-dpdk

DESTBIN=$DESTDIR/bin
install -d -m0755 $DESTBIN
go_install 0755 $DESTBIN ndndpdk-hrlog2histogram
install -m0755 cmd/mgmtclient/mgmtcmd.sh $DESTBIN/ndndpdk-mgmtcmd

NPMTARBALL=$(npm pack -s .)
npm install -C /usr/local -g $NPMTARBALL
rm $NPMTARBALL
