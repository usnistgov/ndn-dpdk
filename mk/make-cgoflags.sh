#!/bin/bash
set -e

if [[ $# -lt 1 ]]; then
  echo 'USAGE: mk/make-cgoflags.sh package libname...' >/dev/stderr
  exit 1
fi

PKG=$(realpath --relative-to=. $1)
PKGNAME=$(basename $PKG)
LIBPATH=$(realpath --relative-to=$PKG build/)
shift

GOFILES=$(find $PKG -maxdepth 1 -name '*.go' -not -name '*_test.go' -not -name 'cgoflags.go')
if [[ -n $GOFILES ]]; then
  PKGNAME=$(grep -h '^package ' $GOFILES | head -1 | awk '{print $2}')
fi

source mk/cflags.sh

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
) | gofmt > $PKG/cgoflags.go
