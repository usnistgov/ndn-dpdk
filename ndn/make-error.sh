#!/bin/bash
set -e
set -o pipefail
cd "$( dirname "${BASH_SOURCE[0]}" )"

(
  echo '#ifndef NDN_DPDK_NDN_ERROR_H'
  echo '#define NDN_DPDK_NDN_ERROR_H'
  echo
  echo '/// \file'
  echo
  echo 'typedef enum NdnError {'
  awk '{ print "  NdnError_" $1 "," }' error.tsv
  echo '} NdnError;'
  echo
  echo '#endif // NDN_DPDK_NDN_ERROR_H'
) > error.h

(
  echo 'package ndn'
  echo
  echo 'import "fmt"'
  echo
  echo 'type NdnError int'
  echo
  echo 'const ('
  awk 'NR == 1 { print "\tNdnError_" $1 " NdnError = iota" }
       NR > 1  { print "\tNdnError_" $1 }' error.tsv
  echo ')'
  echo
  echo 'func (e NdnError) Error() string {'
  echo '  switch e {'
  awk '{ print "  case NdnError_" $1 ": return \"" $1 "\""  }' error.tsv
  echo '  }'
  echo '  return fmt.Sprintf("%d", e)'
  echo '}'
) | gofmt -s > error.go