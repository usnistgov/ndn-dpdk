#!/bin/bash
set -e
set -o pipefail
cd "$( dirname "${BASH_SOURCE[0]}" )"/../..
BPFCC=${BPFCC:-clang-8}
BPFFLAGS='-O2 -target bpf -Wno-int-to-void-pointer-cast -I/usr/include/x86_64-linux-gnu'

mkdir -p build/strategyelf
for F in strategy/*.c; do
  $BPFCC $BPFFLAGS -c $F -o build/strategyelf/$(basename -s .c $F).o
done
go-bindata -nomemcopy -nometadata -pkg strategyelf -prefix build/strategyelf -o strategy/strategyelf/bindata.go build/strategyelf
gofmt -s -w strategy/strategyelf/bindata.go
