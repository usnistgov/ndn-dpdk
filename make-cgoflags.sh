#!/bin/bash
set -e

if [[ $# -lt 1 ]]; then
  echo 'USAGE: ./make-cgoflags.sh package libname...' >/dev/stderr
  exit 1
fi

PKG=$(realpath --relative-to=. $1)
PKGNAME=$(basename $PKG)
LIBPATH=$(realpath --relative-to=$PKG build/)
shift

for GOFILE in $(find $PKG -maxdepth 1 -name '*.go' -not -name '*_test.go' -not -name 'cgoflags.go'); do
  PKGNAME=$(grep 'package ' $GOFILE | head -1 | awk '{print $2}')
done

CFLAGS='-Werror -Wno-error=deprecated-declarations -m64 -pthread -O3 -g -march=native -I/usr/local/include/dpdk'
LIBS='-L/usr/local/lib -lurcu-qsbr -lurcu-cds -ldpdk -ldl -lnuma'
if [[ -n $RELEASE ]]; then
  CFLAGS=$CFLAGS' -DNDEBUG -DZF_LOG_DEF_LEVEL=ZF_LOG_INFO'
fi

(
  echo 'package '$PKGNAME
  echo
  echo '/*'
  echo '#cgo CFLAGS: '$CFLAGS
  echo -n '#cgo LDFLAGS: -L'$LIBPATH
  while [[ -n $1 ]]; do
    DEP=$1
    DEPLIB=$(basename $DEP)
    shift

    echo -n ' -lndn-dpdk-'$DEPLIB
    if [[ -f $DEP/cgoflags.go ]]; then
      for DEPDEP in $(grep LDFLAGS $DEP/cgoflags.go | tr ' ' '\n' | grep 'lndn-dpdk-'); do
        echo -n ' '$DEPDEP
      done
    fi
  done
  echo ' '$LIBS
  echo '*/'
  echo 'import "C"'
) > $PKG/cgoflags.go
