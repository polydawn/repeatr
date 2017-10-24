#!/bin/bash
set -euo pipefail

./cmd.fmt.sh
./cmd.install.sh
VERBOSE=" " ./cmd.test.sh
