#!/bin/bash
set -e
DESTDIR=${DESTDIR:-/usr/local}

DESTSBIN=$DESTDIR/sbin
install -d -m0755 $DESTSBIN
install -m0744 ../../bin/ndnfw-dpdk $DESTSBIN
install -m0744 ../../bin/ndnping-dpdk $DESTSBIN

DESTBIN=$DESTDIR/bin
install -d -m0755 $DESTBIN
install -m0755 cmd/mgmtclient/mgmtcmd.sh $DESTBIN/ndndpdk-mgmtcmd

DESTNODE=$DESTDIR/lib/node_modules/@usnistgov/ndn-dpdk
install -d -m0755 $DESTNODE
find build -name '*.js' | while IFS= read -r SRC; do
  install -D -m0644 $SRC ${SRC/build/$DESTNODE}
done

install -m0644 package.json package-lock.json $DESTNODE
pushd $DESTNODE >/dev/null
npm install --production
chmod -R go-w node_modules
popd >/dev/null

install_node_command() {
  local SCRIPT=$1
  local COMMAND=ndndpdk-$(basename $1)
  local SH=$DESTBIN/$COMMAND
  (
    echo '#!/bin/bash'
    echo '/usr/bin/env node '$DESTNODE/$SCRIPT' "$@"'
  ) >$SH
  chmod 0755 $SH
}

install_node_command cmd/mgmtclient/create-face
