#!/bin/bash
set -e
set -o pipefail
BUILDDIR=$(pwd)
cd "$( dirname "${BASH_SOURCE[0]}" )"/..

mk_cgotest() {
  pushd $1 >/dev/null
  (
    echo 'package '$(basename $1)
    echo 'import "testing"'
    sed -n 's/^func ctest\([^(]*\).*$/func Test\1(t *testing.T) {\nctest\1(t)\n}\n/ p' *_ctest.go
  ) | gofmt -s > cgo_test.go
  popd >/dev/null
}

if [[ $# -lt 1 ]]; then
  echo 'USAGE: mk/cgotest.sh ...package-path' >/dev/stderr
  exit 1
fi

while [[ -n $1 ]]; do
  mk_cgotest $1
  shift
done

touch $BUILDDIR/cgotest.done
