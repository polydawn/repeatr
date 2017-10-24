#!/bin/bash
#
# An interesting thing to do with REPEATR_FLAGS might be:
#   REPEATR_FLAGS='--executor=runc' ./example_runAll.sh
#
mkdir -p wares
for f in example_*.formula ; do echo ==== $f ==== ; repeatr run $REPEATR_FLAGS -- $f ; echo ------- ; echo ; done
