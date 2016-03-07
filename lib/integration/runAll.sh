#!/bin/bash
set -euo pipefail
set -x

# Run each integration example; useful for CI

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$DIR"/../..

for formula in `ls -1 lib/integration/*.json`; do
    echo $formula
    time ./goad exec run $formula
    echo
done
