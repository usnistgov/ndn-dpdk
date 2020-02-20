#!/bin/bash
source mk/cflags.sh
export GODEFCC=$CC
export CC=$PWD/mk/godefcc.sh

cd $(dirname $1)
go tool cgo -godefs -- cgostruct.in.go > cgostruct.go
EXITCODE=$?
rm -rf _obj

if [[ $EXITCODE -eq 0 ]]; then
  gofmt -s -w cgostruct.go
else
  rm cgostruct.go
fi
exit $EXITCODE
