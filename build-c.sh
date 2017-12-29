#!/bin/bash
set -e
INPUTDIR=$1
PKG=$(basename $INPUTDIR)

BUILDDIR=build-c/$PKG
LIBNANE=build-c/libndn-dpdk-$PKG.a
CFLAGS='-m64 -pthread -O3 -march=native -I/usr/local/include/dpdk'

mkdir -p $BUILDDIR
rm -f $BUILDDIR/*.o

for CFILE in $INPUTDIR/*.c; do
  gcc -c -Werror -o $BUILDDIR/$(basename $CFILE .c).o $CFLAGS $CFILE
done
ar rcs $LIBNANE $BUILDDIR/*.o