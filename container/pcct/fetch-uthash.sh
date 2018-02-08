#!/bin/bash
curl -L https://github.com/troydhanson/uthash/raw/v2.0.2/src/uthash.h | sed 's/unsigned hashv;/uint64_t hashv;/' > uthash.h
