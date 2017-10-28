#!/bin/bash
source plugins.shlib
source goprj.preamble.shlib

mkdir -p bin/plugins
mkdir -p .tmp
rio unpack .tmp/runc "$PLUGIN_RUNC_WAREID" --placer=direct --source="$WAREHOUSE_1" --source="$WAREHOUSE_2"
mv .tmp/runc/runc bin/plugins/repeatr-plugin-runc
