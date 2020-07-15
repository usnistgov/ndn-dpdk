#!/bin/bash
set -e
set -o pipefail
BUILDDIR=$(pwd)
cd "$( dirname "${BASH_SOURCE[0]}" )"/..
source mk/cflags.sh

export GODEFCC=$CC
export CC=$PWD/mk/godefcc.sh

mk_cgostruct() {
  pushd $1 >/dev/null
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

if [[ $# -lt 1 ]]; then
  echo 'USAGE: mk/cgostruct.sh ...package-path' >/dev/stderr
  exit 1
fi

while [[ -n $1 ]]; do
  mk_cgostruct $1
  shift
done
