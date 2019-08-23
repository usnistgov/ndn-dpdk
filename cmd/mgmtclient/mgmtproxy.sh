#!/bin/bash
if [[ -z $1 ]]; then
  echo 'Usage: sudo '$0' start|stop [-v]' >/dev/stderr
  exit 2
fi

if [[ $1 == 'start' ]]; then
  socat $2 TCP-LISTEN:6345,reuseaddr,fork,bind=127.0.0.1 \
           UNIX-CONNECT:/var/run/ndn-dpdk-mgmt.sock &
  echo $! > /var/run/ndndpdk-mgmtproxy.pid
fi

if [[ $1 == 'stop' ]]; then
  kill $(cat /var/run/ndndpdk-mgmtproxy.pid)
fi
