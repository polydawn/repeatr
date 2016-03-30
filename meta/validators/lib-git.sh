#!/bin/bash

if [ -z "$VALIDATE_UPSTREAM" ]; then
	# this is kind of an expensive check, so let's not do this twice if we
	# are running more than one validate bundlescript
	
	VALIDATE_REPO='https://github.com/polydawn/repeatr.git'
	VALIDATE_BRANCH='master'
	
	if [ "$TRAVIS" = 'true' -a "$TRAVIS_PULL_REQUEST" != 'false' ]; then
		VALIDATE_REPO="https://github.com/${TRAVIS_REPO_SLUG}.git"
		VALIDATE_BRANCH="${TRAVIS_BRANCH}"
	fi
	
	VALIDATE_HEAD="$(git rev-parse --verify HEAD)"
	if [ -z "$VALIDATE_ALL" ]; then
		VALIDATE_BEGIN="$(git rev-parse --verify "refs/heads/$VALIDATE_BRANCH")"
	else
		VALIDATE_BEGIN="$(git rev-list HEAD | tail -n 1)"
	fi
	VALIDATE_COMMITISH_RANGE="$VALIDATE_BEGIN..$VALIDATE_HEAD"

	validate_diff() {
		if [ "$VALIDATE_BEGIN" != "$VALIDATE_HEAD" ]; then
			git diff "$VALIDATE_COMMITISH_RANGE" "$@"
		fi
	}
	validate_log() {
		if [ "$VALIDATE_BEGIN" != "$VALIDATE_HEAD" ]; then
			git log "$VALIDATE_COMMITISH_RANGE" "$@"
		fi
	}
fi
