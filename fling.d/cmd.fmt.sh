#!/bin/bash
source goprj.preamble.shlib

SUBSECTION=${1:-"..."}
SUBSECTION="./$SUBSECTION"
shift || true
go fmt "$SUBSECTION" "$@"
