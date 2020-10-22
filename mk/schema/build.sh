#!/bin/bash
set -e
set -o pipefail
if [[ -z $MESON_SOURCE_ROOT ]] || [[ -z $MESON_BUILD_ROOT ]] || [[ $# -lt 1 ]]; then
  echo 'USAGE: ninja -C build schema' >/dev/stderr
  exit 1
fi
cd $MESON_SOURCE_ROOT

INFILE=js/types/mod.ts
OUTDIR=$MESON_BUILD_ROOT/share/ndn-dpdk

mkdir -p $OUTDIR
node mk/schema/make-jrgen.js $INFILE Mgmt >$OUTDIR/jsonrpc2.jrgen.json
node mk/schema/make-schema.js $INFILE FaceLocator >$OUTDIR/locator.schema.json
node mk/schema/make-schema.js $INFILE ActivateFwArgs >$OUTDIR/forwarder.schema.json
node mk/schema/make-schema.js $INFILE ActivateGenArgs >$OUTDIR/trafficgen.schema.json
