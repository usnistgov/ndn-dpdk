#!/bin/bash
set -euo pipefail
XDP=$1
if [[ -z $MESON_SOURCE_ROOT ]] || [[ -z $MESON_BUILD_ROOT ]] || [[ -z $XDP ]]; then
  echo 'USAGE: meson compile -C build bpf' >/dev/stderr
  exit 1
fi

BPFCC=${BPFCC:-clang-15}
BPFFLAGS='-g -O2 -target bpf -Wno-int-to-void-pointer-cast -I/usr/include/x86_64-linux-gnu'
BPFDIR=${MESON_BUILD_ROOT}/lib/bpf
mkdir -p "$BPFDIR"

build_category() {
  CATEGORY=$1
  local F
  for F in "${MESON_SOURCE_ROOT}/bpf/${CATEGORY}"/*.c; do
    $BPFCC $BPFFLAGS -c "$F" -o "${BPFDIR}/ndndpdk-${CATEGORY}-$(basename -s .c "$F").o"
  done
}

build_category strategy
if [[ $XDP == 'xdp=1' ]]; then
  build_category xdp
fi
