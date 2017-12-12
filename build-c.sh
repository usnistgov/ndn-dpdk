#!/bin/bash
PKG=$1

BUILDDIR=build-c/$PKG
LIBNANE=build-c/libndn-traffic-dpdk-$PKG.a
CFLAGS='-m64 -pthread -O3 -march=native -I/usr/local/include/dpdk'

mkdir -p $BUILDDIR
rm -f $BUILDDIR/*.o

for CFILE in $PKG/*.c; do
  gcc -c -o $BUILDDIR/$(basename $CFILE .c).o $CFLAGS $CFILE
done
ar rcs $LIBNANE $BUILDDIR/*.o