#!/bin/bash
# Run each integration example; useful for CI
set -euo pipefail
set -x

# CI requires some special modes, but you probably don't on localhost.
CI_FLAGS=${CI_FLAGS:-}

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$DIR"/../..

for formula in `ls -1 lib/integration/*.json`; do
    echo $formula
    time ./goad exec run $CI_FLAGS $formula
    echo
done
