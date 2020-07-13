#!/bin/bash
set -e
set -o pipefail

(
  grep -E '^package ' test_test.go | head -1
  echo 'import "testing"'
  sed -n 's/^func ctest\([^(]*\).*$/func Test\1(t *testing.T) {\nctest\1(t)\n}\n/ p' *_ctest.go
) | gofmt -s > cgo_test.go
