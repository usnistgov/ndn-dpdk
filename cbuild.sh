#!/bin/bash
set -e

if [[ $# -ne 1 ]]; then
  echo 'USAGE: ./cbuild.sh [directory|file]' >/dev/stderr
  exit 1
fi

CFLAGS='-Werror -Wno-error=deprecated-declarations -m64 -pthread -O3 -g -march=native -I/usr/local/include/dpdk -I/usr/include/dpdk'

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
LIBNAME=build/libndn-dpdk-$PKG.a

mkdir -p $BUILDDIR
rm -f $LIBNAME $BUILDDIR/*.o

ar rc $LIBNAME
for CFILE in $(find $INPUTDIR -maxdepth 1 -name '*.c'); do
  OBJ=$BUILDDIR/$(basename $CFILE .c).o
  gcc -c -Werror -o $OBJ $CFLAGS $CFILE
  ar r $LIBNAME $OBJ
done
ar s $LIBNAME
