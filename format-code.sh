#!/bin/bash
go fmt ./...
find -name '*.h' -o -name '*.c' \
  | grep -vE 'pcg_basic|siphash-20121104|uthash|zf_log' \
  | xargs clang-format -i -style='{BasedOnStyle: Mozilla, ReflowComments: false}'
