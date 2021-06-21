#!/bin/bash
set -eo pipefail
if [[ -z $MESON_SOURCE_ROOT ]] || [[ -z $MESON_BUILD_ROOT ]] || [[ $# -lt 1 ]]; then
  echo 'USAGE: ninja -C build cgotest' >/dev/stderr
  exit 1
fi
cd $MESON_SOURCE_ROOT

mk_cgotest() {
  pushd $1 >/dev/null
  (
    echo 'package '$(basename $1)
    echo 'import "testing"'
    sed -nE \
      -e 's/^func ctest([^(]*).*$/func Test\1(t *testing.T) {\nctest\1(t)\n}\n/ p' \
      -e 's/^func cbench([^(]*).*$/func Bench\1(b *testing.B) {\ncbench\1(b)\n}\n/ p' \
      *_ctest.go
  ) | gofmt -s > cgo_test.go
  popd >/dev/null
}

while [[ -n $1 ]]; do
  mk_cgotest $1
  shift
done
