#!/bin/bash

getTestPkg() {
  # determine $TESTPKG from $PKG
  if [[ $1 == 'app/fwdp' ]]; then echo app/fwdp/fwdptest
  elif [[ $1 == 'container/cs' ]]; then echo container/cs/cstest
  elif [[ $1 == 'container/fib' ]]; then echo container/fib/fibtest
  elif [[ $1 == 'container/mintmr' ]]; then echo container/mintmr/mintmrtest
  elif [[ $1 == 'core' ]]; then echo core/coretest
  elif [[ $1 == 'dpdk' ]]; then echo dpdk/dpdktest
  elif [[ $1 == 'iface' ]]; then echo iface/ifacetest
  else echo $PKG; fi
}

if [[ $# -eq 0 ]]; then
  # run all tests

  find -name '*_test.go' -printf '%h\n' | uniq | xargs -I{} sudo $(which go) test {}

elif [[ $# -eq 1 ]]; then
  # run tests in one package
  PKG=${1%/}
  TESTPKG=$(getTestPkg $PKG)

  sudo $(which go) test -cover -covermode count -coverpkg ./$PKG -coverprofile /tmp/gotest.cover ./$TESTPKG -v
  go tool cover -html /tmp/gotest.cover -o /tmp/gotest.cover.html

elif [[ $# -eq 2 ]]; then
  # run one test
  PKG=${1%/}
  TESTPKG=$(getTestPkg $PKG)
  TEST=$2

  sudo GODEBUG=cgocheck=2 $DBG $(which go) test ./$TESTPKG -v -run 'Test'$TEST'.*'

elif [[ $# -eq 3 ]]; then
  # run one test with debug tool
  DBGTOOL=$1
  PKG=${2%/}
  TESTPKG=$(getTestPkg $PKG)
  TEST=$3

  if [[ $DBGTOOL == 'gdb' ]]; then DBG='gdb --args'
  elif [[ $DBGTOOL == 'valgrind' ]]; then DBG='valgrind'
  else
    echo 'Unknown debug tool:' $1 >/dev/stderr
    exit 1
  fi

  go test -c ./$TESTPKG -o /tmp/gotest.exe
  sudo $DBG /tmp/gotest.exe -test.v -test.run 'Test'$TEST'.*'
else
  echo 'USAGE: ./gotest.sh [debug-tool] [directory] [test-name]' >/dev/stderr
  exit 1
fi
