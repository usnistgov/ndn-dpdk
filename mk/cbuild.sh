#!/bin/bash
set -e
set -o pipefail
source mk/cflags.sh

if [[ $# -ne 1 ]]; then
  echo 'USAGE: mk/cbuild.sh [directory|file]' >/dev/stderr
  exit 1
fi

if [[ -f $1 ]]; then
  CFILE=$1
  $CC -c $CFLAGS -o /dev/null $CFILE
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
for CFILE in $(find $INPUTDIR -maxdepth 1 -name '*.c' | sort); do
  OBJ=$BUILDDIR/$(basename $CFILE .c).o
  $CC -c -Werror -o $OBJ $CFLAGS $CFILE
  ar r $LIBNAME $OBJ
done
ar s $LIBNAME
