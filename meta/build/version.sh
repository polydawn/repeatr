#!/bin/bash
#
# Source this to append the LD_FLAGS var with magic to set version info.
#
# Export GITCOMMIT and BUILDDATE to override the autodetection
# (and eschew the need to invoke git).
#

GITCOMMIT=${GITCOMMIT:-}
if [ -z "$GITCOMMIT" ]; then
	GITCOMMIT="$(git rev-parse --short HEAD)"
	if [ -n "$(git status --porcelain --untracked-files=no)" ]; then
		GITCOMMIT="$GITCOMMIT-dirty"
	fi
fi

BUILDDATE="${BUILDDATE:-$(date --rfc-2822)}"

LDFLAGS+=" -X '$cmd/version.GitCommit=$GITCOMMIT'"
LDFLAGS+=" -X '$cmd/version.BuildDate=$BUILDDATE'"
