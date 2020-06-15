#!/bin/bash
set -e
set -o pipefail

if [[ $# -lt 1 ]]; then
  echo 'USAGE: mk/cgoflags.sh package' >/dev/stderr
  exit 1
fi

PKG=$(realpath --relative-to=. $1)
PKGNAME=$(basename $PKG)
BUILDDIR=${BUILDDIR:-build}
LIBPATH=$(realpath --relative-to=$PKG $BUILDDIR/)

GOFILES=$(find $PKG -maxdepth 1 -name '*.go' -not -name '*_test.go' -not -name 'cgoflags.go')
if [[ -n $GOFILES ]]; then
  PKGNAME=$(grep -h '^package ' $GOFILES | head -1 | awk '{print $2}')
fi

MK_CGOFLAGS=1
source mk/cflags.sh

(
  echo 'package '$PKGNAME
  echo
  echo '/*'
  echo '#cgo CFLAGS: '$CGO_CFLAGS
  echo '#cgo LDFLAGS: -L'$LIBPATH' -lndn-dpdk-c '$CGO_LIBS
  echo '*/'
  echo 'import "C"'
) | gofmt -s > $PKG/cgoflags.go
