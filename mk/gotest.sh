#!/bin/bash
set -e
set -o pipefail
source mk/cflags.sh

getTestPkg() {
  # determine $TESTPKG from $PKG
  TESTDIR=$1/$(basename $1)test
  if [[ -d $TESTDIR ]]; then
    echo $TESTDIR
  else
    echo $1
  fi
}

if [[ $# -eq 0 ]]; then
  # run all tests, optional filter in $MK_GOTEST_FILTER

  find -name '*_test.go' -printf '%h\n' | uniq | sed -E "${MK_GOTEST_FILTER:-}" \
    | xargs -I{} sudo -E $(which go) test {} -count=1

elif [[ $# -eq 1 ]]; then
  # run tests in one package
  PKG=${1%/}
  TESTPKG=$(getTestPkg $PKG)

  sudo -E $(which go) test -cover -covermode count -coverpkg ./$PKG -coverprofile /tmp/gotest.cover ./$TESTPKG -v
  sudo chown $(id -u) /tmp/gotest.cover
  go tool cover -html /tmp/gotest.cover -o /tmp/gotest.cover.html

elif [[ $# -eq 2 ]]; then
  # run one test
  PKG=${1%/}
  TESTPKG=$(getTestPkg $PKG)
  TEST=$2

  sudo -E GODEBUG=cgocheck=2 $DBG $(which go) test ./$TESTPKG -count=1 -v -run 'Test'$TEST'.*'

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
  sudo -E $DBG /tmp/gotest.exe -test.v -test.run 'Test'$TEST'.*'
else
  echo 'USAGE: mk/gotest.sh [debug-tool] [directory] [test-name]' >/dev/stderr
  exit 1
fi
