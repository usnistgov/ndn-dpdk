#!/bin/bash
if [[ $# -eq 1 ]]; then
  sudo $(which go) test ./$1 -v
else
  sudo $(which go) test ./$1 -v -run 'Test'$2'.*'
fi