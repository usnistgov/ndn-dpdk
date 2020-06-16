#!/bin/bash
set -e
set -o pipefail

TOOLPATH=/tmp/ndn-dpdk-gogenerate
export PATH=$TOOLPATH/bin:$PATH

if ! which stringer >/dev/null; then
  GO111MODULE=off GOPATH=$TOOLPATH go get golang.org/x/tools/cmd/stringer
fi

PKG=$1
if [[ -z $PKG ]]; then
  PKG=./...
fi
go generate $PKG
