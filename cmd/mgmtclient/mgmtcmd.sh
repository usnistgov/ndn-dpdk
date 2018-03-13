#!/bin/bash
if [[ $1 == 'help' ]]; then
  cat <<EOT
Subcommands:
  face [list]
    List faces.
  face show <ID>
    Show face counters.
  dataplane counters
    Show dataplane counters.
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
elif [[ $1 == 'dataplane' ]]; then
  if [[ $2 == 'counters' ]]; then
    jsonrpc DataPlane.GetCounters ''
  fi
fi