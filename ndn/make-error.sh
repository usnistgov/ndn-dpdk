#!/bin/bash
cd "$( dirname "${BASH_SOURCE[0]}" )"

(
  echo '#ifndef NDN_DPDK_NDN_ERROR_H'
  echo '#define NDN_DPDK_NDN_ERROR_H'
  echo
  echo 'typedef enum NdnError {'
  awk '{ print "  NdnError_" $1 "," }' error.tsv
  echo '} NdnError;'
  echo
  echo '#endif // NDN_DPDK_NDN_ERROR_H'
) > error.h

awk '
BEGIN {
  print "package ndn"
  print ""
  print "import \"fmt\""
  print ""
  print "type NdnError int"
  print ""
  print "func (e NdnError) Error() string {"
  print "  return fmt.Sprintf(\"NDN error code %d\", e)"
  print "}"
  print ""
  print "const ("
}
NR == 1 {
  print "\tNdnError_" $1 " NdnError = iota"
}
NR > 1 {
  print "\tNdnError_" $1
}
END {
  print ")"
}
' error.tsv > error.go
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
) | gofmt > error.go