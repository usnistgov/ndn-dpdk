#!/bin/bash
set -eo pipefail

while ! nfdc status; do
  sleep 1
done