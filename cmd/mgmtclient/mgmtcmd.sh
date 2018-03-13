#!/bin/bash
if [[ $1 == 'help' ]]; then
  cat <<EOT
Subcommands:
  face [list]
    List faces.
  face show <ID>
    Show face counters.
  dpinfo [global]
    Show dataplane global information.
  dpinfo input <I>
  dpinfo fwd <I>
  dpinfo pit <I>
  dpinfo cs <I>
    Show dataplane i-th input/fwd/PIT/CS counters.
EOT
  exit 0
fi

jsonrpc() {
  METHOD=$1
  PARAMS=$2
  if [[ -z $2 ]]; then PARAMS='{}'; fi
  jayson -s 127.0.0.1:6345 -m $METHOD -p "$PARAMS"
}

if [[ $1 == 'face' ]]; then
  if [[ -z $2 ]] || [[ $2 == 'list' ]]; then
    jsonrpc Faces.List
  elif [[ $2 == 'show' ]]; then
    jsonrpc Faces.Get '{"Id":'$3'}'
  fi
elif [[ $1 == 'dpinfo' ]]; then
  if [[ -z $2 ]] || [[ $2 == 'global' ]]; then
    jsonrpc DPInfo.Global ''
  elif [[ $2 == 'input' ]] || [[ $2 == 'fwd' ]] || [[ $2 == 'pit' ]] || [[ $2 == 'cs' ]]; then
    jsonrpc DPInfo."${2^}" '{"Index":'$3'}'
  fi
fi
