#!/bin/bash
go fmt ./...
find -name '*.h' -o -name '*.c' \
  | grep -v core/zf_log \
  | xargs clang-format -i -style='{BasedOnStyle: Mozilla, ReflowComments: false}'