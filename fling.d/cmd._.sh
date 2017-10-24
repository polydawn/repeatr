#!/bin/bash
set -euo pipefail

./cmd.install-deps.sh
./cmd.fmt.sh
./cmd.install.sh
VERBOSE=" " ./cmd.test.sh
