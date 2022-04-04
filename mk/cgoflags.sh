#!/bin/bash
set -euo pipefail
if [[ -z $MESON_SOURCE_ROOT ]] || [[ -z $MESON_BUILD_ROOT ]] || [[ $# -lt 1 ]]; then
  echo 'USAGE: ninja -C build cgoflags' >/dev/stderr
  exit 1
fi
cd "$MESON_SOURCE_ROOT"

MK_CGOFLAGS=1
source mk/cflags.sh

mk_cgoflags() {
  local PKG=$(realpath --relative-to=. "$1")
  local PKGNAME=$(basename "$PKG")
  local LIBPATH=$(realpath --relative-to="$PKG" "$MESON_BUILD_ROOT/")

  GOFILES=$(find "$PKG" -maxdepth 1 -name '*.go' -not -name '*_test.go' -not -name 'cgoflags.go')
  if [[ -n $GOFILES ]]; then
    PKGNAME=$(grep -h '^package ' $GOFILES | head -n1 | awk '{print $2}')
  fi

  gofmt -s > "$PKG"/cgoflags.go <<EOF
package $PKGNAME

/*
#cgo CFLAGS: $CGO_CFLAGS
#cgo LDFLAGS: -L$LIBPATH -lndn-dpdk-c $CGO_LIBS
*/
import "C"
EOF
}

for D do
  mk_cgoflags "$D"
done
