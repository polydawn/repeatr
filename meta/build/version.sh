#!/bin/bash
#
# Source this to append the LD_FLAGS var with magic to set version info.
#
# Export GITCOMMIT and BUILDDATE to override the autodetection
# (and eschew the need to invoke git).
#

if [ -z "$(which git)" ]; then
	GITCOMMIT=${GITCOMMIT:-'!!Unknown!!'}
	GITDIRTY=${GITDIRTY:-'!!Unknown!!'}
	TREEHASH=${TREEHASH:-'!!Unknown!!'}
	COMMITDATE=${COMMITDATE:-'!!Unknown!!'}
	AUTHORDATE=${AUTHORDATE:-'!!Unknown!!'}
else
	GITCOMMIT=${GITCOMMIT:-}
	GITDIRTY=${GITDIRTY:-}
	if [ -z "$GITCOMMIT" ]; then
		GITCOMMIT="$(git rev-parse HEAD)"
		if [ -n "$(git status --porcelain)" ]; then
			GITCOMMIT="$GITCOMMIT"
			GITDIRTY="true"
		else
			GITDIRTY="false"
		fi
	fi

	TREEHASH=${TREEHASH:-}
	if [ -z "$TREEHASH" ]; then
		TREEHASH=$(
			export GIT_INDEX_FILE=$(mktemp)
			git read-tree HEAD # Required for the index file to be usable
			git add -A
			git write-tree # Print a hash of all that is indexed. Amen.
		)
	fi

	COMMITDATE=${COMMITDATE:-}
	if [ -z "$COMMITDATE" ]; then
		COMMITDATE="$(date -R --utc --date="$(git log -1 --format=%cd --date=rfc2822 ${GITCOMMIT})")"
	fi

	AUTHORDATE=${AUTHORDATE:-}
	if [ -z "$AUTHORDATE" ]; then
		AUTHORDATE="$(date -R --utc --date="$(git log -1 --format=%ad --date=rfc2822 ${GITCOMMIT})")"
	fi
fi


LDFLAGS+=" -X '$cmd/version.GitCommit=$GITCOMMIT'"
LDFLAGS+=" -X '$cmd/version.GitDirty=$GITDIRTY'"
LDFLAGS+=" -X '$cmd/version.GitTreeHash=$TREEHASH'"
LDFLAGS+=" -X '$cmd/version.GitCommitDate=$COMMITDATE'"
LDFLAGS+=" -X '$cmd/version.GitAuthorDate=$AUTHORDATE'"
