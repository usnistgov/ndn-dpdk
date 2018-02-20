#!/bin/bash
set -e

if [[ $# -ne 1 ]]; then
  echo 'USAGE: ./cbuild.sh [directory|file]' >/dev/stderr
  exit 1
fi

CFLAGS='-Werror -m64 -pthread -O3 -g -march=native -I/usr/local/include/dpdk'

if [[ -f $1 ]]; then
  CFILE=$1
  gcc -c $CFLAGS -o /dev/null $CFILE
  exit
fi

if ! [[ -d $1 ]]; then
  echo 'Directory '$1' not found.' >/dev/stderr
  exit 1
fi

INPUTDIR=$(realpath --relative-to=. $1)
PKG=$(basename $INPUTDIR)
BUILDDIR=build/$PKG
LIBNANE=build/libndn-dpdk-$PKG.a

mkdir -p $BUILDDIR
rm -f $BUILDDIR/*.o

for CFILE in $INPUTDIR/*.c; do
  gcc -c -Werror -o $BUILDDIR/$(basename $CFILE .c).o $CFLAGS $CFILE
done
ar rcs $LIBNANE $BUILDDIR/*.o
