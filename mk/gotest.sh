#!/bin/bash
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"/..
TESTCOUNT=${TESTCOUNT:-1}

SUDO='sudo -E'
if [[ $(id -u) -eq 0 ]]; then
  SUDO=
fi

getTestPkg() {
  # determine $TESTPKG from $PKG
  TESTDIR=$1/$(basename "$1")test
  if [[ -d $TESTDIR ]]; then
    echo "$TESTDIR"
  else
    echo "$1"
  fi
}

if [[ $# -eq 0 ]]; then
  # run all tests, optional filter in $MK_GOTEST_FILTER
  find -name '*_test.go' -printf '%h\n' | sort -u | sed -E "${MK_GOTEST_FILTER:-}" |
    xargs -I{} $SUDO mk/go.sh test {} -count=$TESTCOUNT

elif [[ $# -eq 1 ]]; then
  # run tests in one package
  PKG=${1%/}
  TESTPKG=$(getTestPkg "$PKG")
  COVERPKG=./"$PKG"
  case $PKG in
    container/cs) COVERPKG=$COVERPKG,./container/pcct ;;
    container/fib) COVERPKG=$COVERPKG,./container/fib/... ;;
    container/pit) COVERPKG=$COVERPKG,./container/pcct ;;
    iface/ethface) COVERPKG=$COVERPKG,./iface/ethport ;;
  esac

  $SUDO rm -f /tmp/gotest.cover
  $SUDO mk/go.sh test -cover -covermode count -coverpkg "$COVERPKG" -coverprofile /tmp/gotest.cover ./"$TESTPKG" -v -count=$TESTCOUNT
  $SUDO chown "$(id -u)" /tmp/gotest.cover
  mk/go.sh tool cover -html /tmp/gotest.cover -o /tmp/gotest.cover.html

elif [[ $# -eq 2 ]]; then
  # run one test
  PKG=${1%/}
  TESTPKG=$(getTestPkg "$PKG")
  TEST=$2

  if [[ ${TEST,,} != test* ]] && [[ ${TEST,,} != bench* ]] && [[ ${TEST,,} != example* ]]; then
    TEST=Test$TEST
  fi
  RUN=(-run "$TEST")
  if [[ ${TEST,,} == bench* ]]; then
    RUN+=(-bench "$TEST")
  fi
  $SUDO env GOEXPERIMENT=cgocheck2 mk/go.sh test ./"$TESTPKG" -v -count=$TESTCOUNT "${RUN[@]}"

elif [[ $# -eq 3 ]]; then
  # run one test with debug tool
  DBGTOOL=$1
  PKG=${2%/}
  TESTPKG=$(getTestPkg "$PKG")
  TEST=$3

  case $DBGTOOL in
    gdb) DBG='gdb --silent --args' ;;
    valgrind) DBG='valgrind' ;;
    *)
      echo "Unknown debug tool: $DBGTOOL" >/dev/stderr
      exit 1
      ;;
  esac

  mk/go.sh test -c ./"$TESTPKG" -o /tmp/gotest.exe
  $SUDO $DBG /tmp/gotest.exe -test.v -test.run "Test${TEST}.*"

else
  echo 'USAGE: mk/gotest.sh [debug-tool] [directory] [test-name]' >/dev/stderr
  exit 1
fi
