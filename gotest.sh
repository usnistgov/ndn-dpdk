#!/bin/bash
if [[ $# -eq 0 ]]; then
  find -name '*_test.go' -printf '%h\n' | uniq | xargs -I{} sudo $(which go) test {} -v
elif [[ $# -eq 1 ]]; then
  sudo $(which go) test -cover ./$1 -v
elif [[ $# -eq 2 ]]; then
  sudo GODEBUG=cgocheck=2 $DBG $(which go) test ./$1 -v -run 'Test'$2'.*'
elif [[ $# -eq 3 ]]; then
  if [[ $1 == 'gdb' ]]; then
    DBG='gdb --args'
  elif [[ $1 == 'valgrind' ]]; then
    DBG='valgrind'
  else
    echo 'Unknown debug tool:' $1 >/dev/stderr
    exit 1
  fi
  go test -c ./$2 -o /tmp/gotest-exe
  sudo $DBG /tmp/gotest-exe -test.v -test.run 'Test'$3'.*'
else
  echo 'USAGE: ./gotest.sh [debug-tool] [directory] [test-name]' >/dev/stderr
  exit 1
fi
