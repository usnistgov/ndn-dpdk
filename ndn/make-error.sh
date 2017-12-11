#!/bin/bash
cd "$( dirname "${BASH_SOURCE[0]}" )"

awk '
BEGIN {
  print "#ifndef NDN_TRAFFIC_DPDK_NDN_ERROR_H"
  print "#define NDN_TRAFFIC_DPDK_NDN_ERROR_H"
  print ""
  print "typedef enum {"
}
{
  print "  NdnError_" $1 ","
}
END {
  print "} NdnError;"
  print ""
  print "#endif // NDN_TRAFFIC_DPDK_NDN_ERROR_H"
}
' error.tsv > error.h

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