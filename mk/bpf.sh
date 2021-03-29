#!/bin/bash
set -eo pipefail
if [[ -z $MESON_SOURCE_ROOT ]] || [[ -z $MESON_BUILD_ROOT ]]; then
  echo 'USAGE: ninja -C build cgoflags' >/dev/stderr
  exit 1
fi
cd ${MESON_SOURCE_ROOT}/bpf

BPFCC=${BPFCC:-clang-8}
BPFFLAGS='-O2 -target bpf -Wno-int-to-void-pointer-cast -I/usr/include/x86_64-linux-gnu'
BPFDIR=${MESON_BUILD_ROOT}/lib/bpf

mkdir -p $BPFDIR

for F in strategy/*.c; do
  $BPFCC $BPFFLAGS -c $F -o $BPFDIR/ndndpdk-strategy-$(basename -s .c $F).o
done
