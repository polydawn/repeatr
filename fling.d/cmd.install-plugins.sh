#!/bin/bash
source plugins.shlib
source goprj.preamble.shlib

mkdir -p bin/plugins
mkdir -p .tmp
rio unpack "$PLUGIN_RUNC_WAREID" .tmp/runc --placer=direct --source="$WAREHOUSE_1" --source="$WAREHOUSE_2"
mv .tmp/runc/runc bin/plugins/repeatr-plugin-runc
