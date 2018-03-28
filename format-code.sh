#!/bin/bash
go fmt ./...
find -name '*.h' -o -name '*.c' \
  | grep -vE 'core/zf_log|siphash-20121104|uthash' \
  | xargs clang-format -i -style='{BasedOnStyle: Mozilla, ReflowComments: false}'
