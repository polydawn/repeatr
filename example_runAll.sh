#!/bin/bash
mkdir -p wares
for f in example_* ; do echo ==== $f ==== ; repeatr run $f ; echo ------- ; echo ; done
