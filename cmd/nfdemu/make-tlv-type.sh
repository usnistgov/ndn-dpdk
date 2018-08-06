#!/bin/bash
cd "$( dirname "${BASH_SOURCE[0]}" )"

(
  echo 'export enum TT {'
  awk  'NF==2 { print "  " $1 " = 0x" $2 "," }' ../../ndn/tlv-type.tsv
  echo '}'
) > tlv-type.ts
