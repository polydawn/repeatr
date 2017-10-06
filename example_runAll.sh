#!/bin/bash
mkdir -p wares
for f in example_*.formula ; do echo ==== $f ==== ; repeatr run $f ; echo ------- ; echo ; done
