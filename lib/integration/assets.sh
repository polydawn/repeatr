#!/bin/bash -e

# A very silly way to resolve some testing assets locally.
# Improvements or replacements welcome.

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$DIR"/../..

# Holds convenient downloads. Considered disposable. Can be cached by your CI.
mkdir -p assets

# Assets really ought to be in a DFS?
# For now, canonical URL and mandated URL pattern for maximum portability.
mirrorURL="http://storage.googleapis.com/scitran-dist/assets"

# Takes a file and a sha384
resolveAsset() {
	file="assets/"$1
	hash=$2

	if [[ ! -f $file ]]; then
		# URL pattern is mirror/hash-filename
		# Attempt to ensure an aborted download is deleted
		wget $mirrorURL/$hash-$1 -O $file || rm $file

		# Confirm checksum
		sha384sum $file | grep "$hash " || ( echo "ERROR DOWNLOADING $file - HASH MISMATCH"; rm $file; exit 1 )
	fi
}

# Basic ubuntu container; trivially cleaned from a Docker mirror
resolveAsset "ubuntu.tar.gz" "0b6c6d8318dd8906397384f0a16c01754c29521124abbb6f67886caaec2eb2c3953f8c00dc5cba3734bd7bbae51b007b"

# Nsinit binary; to be upgraded later
resolveAsset "nsinit" "f48447a3e3d44dc94e04c25aac61b979d4a24a894fcd83453fe79a294c04d5a0f4a338d3fba43dd7eb2db5e3e429479a"
chmod +x assets/nsinit


echo "Assets downloaded."
