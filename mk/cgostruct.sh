#!/bin/bash
set -euo pipefail
if [[ -z $MESON_SOURCE_ROOT ]] || [[ -z $MESON_BUILD_ROOT ]] || [[ $# -lt 1 ]]; then
  echo 'USAGE: meson compile -C build cgostruct' >/dev/stderr
  exit 1
fi
cd "$MESON_SOURCE_ROOT"
source mk/cflags.sh

export GODEFCC=$CC
export CC=$PWD/mk/godefcc.sh

mk_cgostruct() {
  pushd "$1" >/dev/null
  set +e
  go tool cgo -godefs -- cgostruct.in.go > cgostruct.go
  EXITCODE=$?
  set -e
  rm -rf _obj

  if [[ $EXITCODE -eq 0 ]]; then
    gofmt -s -w cgostruct.go
  else
    rm cgostruct.go
  fi
  popd >/dev/null
  return $EXITCODE
}

for D do
  mk_cgostruct "$D"
done
