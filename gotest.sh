#!/bin/bash
if [[ $# -eq 0 ]]; then
  find -name '*_test.go' -printf '%h\n' | uniq | xargs -I{} sudo $(which go) test {} -v
elif [[ $# -eq 1 ]]; then
  sudo $(which go) test -cover ./$1 -v
elif [[ $# -eq 2 ]]; then
  sudo GODEBUG=cgocheck=2 $(which go) test ./$1 -v -run 'Test'$2'.*'
else
  echo 'USAGE: ./gotest.sh [directory] [test-name]' >/dev/stderr
  exit 1
fi
