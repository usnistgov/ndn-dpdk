#!/bin/bash
cd "$( dirname "${BASH_SOURCE[0]}" )"

(
  echo '#ifndef NDN_TRAFFIC_DPDK_NDN_TLV_TYPE_H'
  echo '#define NDN_TRAFFIC_DPDK_NDN_TLV_TYPE_H'
  echo
  echo 'typedef enum TlvType {'
  awk '{ print "  TT_" $1 " = 0x" $2 "," }' tlv-type.tsv
  echo '} TlvType;'
  echo
  echo '#endif // NDN_TRAFFIC_DPDK_NDN_TLV_TYPE_H'
) > tlv-type.h

(
  echo 'package ndn'
  echo
  echo 'import "fmt"'
  echo
  echo 'type TlvType uint64'
  echo
  echo 'const ('
  awk '{ print "\tTT_" $1 " TlvType = 0x" $2  }' tlv-type.tsv
  echo ')'
  echo
  echo 'func (tt TlvType) String() string {'
  echo -e '\tswitch tt {'
  awk '{ print "\tcase TT_" $1 ": return \"" $2 "\""  }' tlv-type.tsv
  echo -e '\t}'
  echo -e '\treturn fmt.Sprintf("%d", tt)'
  echo '}'
) | gofmt > tlv-type.go