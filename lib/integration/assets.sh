#!/bin/bash
#
# Bootstrap assets for testing onto the local filesystem.
#
# Test assets bootstrap takes (unusually!) non-CA/DFS coordinates.
# Many parts of testing and demos do use repeatr's own full content-addressible systems, but
# these assets are supposed to work in basic tests at the bottom of the stack even when all
# else fails, so they get special treatment.
#
set -eo pipefail

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$DIR"/../..

# Holds convenient downloads. Considered disposable. Can be cached by your CI.
mkdir -p assets

mirrorURL="http://repeatr.s3.amazonaws.com/custom"

# Takes a file and a sha384
resolveAsset() {
	file="assets/"$1
	hash=$2

	if [[ ! -f $file ]]; then
		# URL pattern is mirror/hash-filename
		# Attempt to ensure an aborted download is deleted
		wget $mirrorURL/$hash-$1 -O $file || rm $file

		# Confirm checksum
		sha384sum $file | grep "$hash " || ( echo "ERROR DOWNLOADING $file - HASH MISMATCH"; rm -f $file; exit 1 )
	fi
}

# Basic ubuntu container; trivially cleaned from a Docker mirror
resolveAsset "ubuntu.tar.gz" "0b6c6d8318dd8906397384f0a16c01754c29521124abbb6f67886caaec2eb2c3953f8c00dc5cba3734bd7bbae51b007b"


echo "Assets downloaded."
