#!/bin/bash
set -euo pipefail
DIR=$1

mkdir -p $DIR
cd $DIR

fallocate -xl 32G F
mkdir T
for I in {0..1023}; do ln -s ../F T/$I; done
for I in {0..11}; do ln -s T $I; done
