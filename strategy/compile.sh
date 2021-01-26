#!/bin/bash
set -eo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"/..
BPFCC=${BPFCC:-clang-8}
BPFFLAGS='-O2 -target bpf -Wno-int-to-void-pointer-cast -I/usr/include/x86_64-linux-gnu'
BPFDIR=build/lib/bpf

mkdir -p $BPFDIR
for F in strategy/*.c; do
  $BPFCC $BPFFLAGS -c $F -o $BPFDIR/ndndpdk-strategy-$(basename -s .c $F).o
done
touch build/strategy.done
