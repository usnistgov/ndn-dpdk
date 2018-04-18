#!/bin/bash
if [[ $1 == 'help' ]]; then
  cat <<EOT
Subcommands:
  face [list]
    List faces.
  face show <ID>
    Show face counters.
  fib info
    Show FIB counters.
  fib list
    List FIB entry names.
  fib insert <NAME> <NEXTHOP,NEXTHOP>
    Insert/replace FIB entry.
  fib erase <NAME>
    Erase FIB entry.
  fib find <NAME>
  fib lpm <NAME>
    Perform exact-match/longest-prefix-match lookup on FIB.
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
    jsonrpc Face.List
  elif [[ $2 == 'show' ]]; then
    jsonrpc Face.Get '{"Id":'$3'}'
  fi
elif [[ $1 == 'fib' ]]; then
  if [[ $2 == 'info' ]]; then
    jsonrpc Fib.Info ''
  elif [[ -z $2 ]] || [[ $2 == 'list' ]]; then
    jsonrpc Fib.List ''
  elif [[ $2 == 'insert' ]]; then
    jsonrpc Fib.Insert '{"Name":"'$3'","Nexthops":['$4']}'
  elif [[ $2 == 'erase' ]] || [[ $2 == 'find' ]] || [[ $2 == 'lpm' ]]; then
    jsonrpc Fib."${2^}" '{"Name":"'$3'"}'
  fi
elif [[ $1 == 'dpinfo' ]]; then
  if [[ -z $2 ]] || [[ $2 == 'global' ]]; then
    jsonrpc DpInfo.Global ''
  elif [[ $2 == 'input' ]] || [[ $2 == 'fwd' ]] || [[ $2 == 'pit' ]] || [[ $2 == 'cs' ]]; then
    jsonrpc DpInfo."${2^}" '{"Index":'$3'}'
  fi
fi
