#!/bin/bash
##
## toolstrap: bootstrap your toolchain
##
## Fetches a thing and puts it in a dir.
## The each version label gets its own dir so that you can download more things
## over time, but they don't fight for space or overwrite each other (useful when
## jumping across git branches that have different requirements, for example).
##
## Why use this instead of wget/curl/hash yourself?  Toolstrap avoids the
## following common pitfalls:
##  - Toolstrap's dir structure avoids collisions
##  - Toolstrap produces nice output on hash mismatch
##  - Toolstrap manages atomicity -- no partial download problems
##  - Toolstrap manages atomicity -- no partial extraction problems
##
## To use toolstrap in your project, copy it into your own git repo,
## source it into your own scripts,
## and call the toolstrap function from there.
##
## The dir structure comes out like this:
##  tools/${tool_name}/${version_name}/*
## Note that this does not include the hash you used to download it; the cache
## is *not* content addressible, and you can easily manually break it.
##
## Hopefully you won't need this for more than once, of course ;)
## Once you have Repeatr, it can do a better job of managing CAS, dedup,
## multiple mirrors, network transparency, etc etc for you :)
##

## USAGE EXAMPLE:
##
## ```
##	mkdir -p tools/
##	toolstrap \
##		repeatr \
##		v0.12 \
##		61ef917c7988d985629a4818858dbc614cb7a6da6c37c2a6bcf6cf97781fc5c83f028243d4c11a2b7d958a1c78fa6c6b \
##	 	https://github.com/polydawn/repeatr/releases/download/release%2Fv0.12/repeatr-linux-amd64-v0.12.tar.gz
## ```
##
## The downloaded file will be extracted into
##  ./tools/repeatr/v0.12/*


toolstrap() {(
	set -euo pipefail

	NAME="$1"
	VERSIONLABEL="$2"
	HASH="$3"
	URL="$4"

	tmpdl="tools/${NAME}/.tmp.dl.${VERSIONLABEL}"
	tmpdir="tools/${NAME}/.tmp.unpack.${VERSIONLABEL}"
	destdir="tools/${NAME}/${VERSIONLABEL}"

	### If there's already a thing in place, early exit.
	if [ -d "${destdir}" ]; then
		echo "$destdir already exists; assuming valid" 1>&2; return 0;
	fi

	### Ensure landing zone; clean up any previous half-attempts.
	mkdir -p "tools/${NAME}/"
	rm -rf "${tmpdl}" "${tmpdir}" || true;

	### Download and check hash against expectation.
	# (This eschews use of '--check' options so it can produce better output,
	# specifically including the actual hash in the case of the check failing.)
	wget -O "${tmpdl}" "${URL}"
	sha384sum "${tmpdl}" | tee /dev/fd/2 | grep ^"${HASH} " >/dev/null \
		|| { echo "corrupt or hash mismatched ${NAME}-${VERSIONLABEL} download" 1>&2; return 16; }

	### Unpack to temp dir (just for atomicity's sake)
	mkdir "${tmpdir}"
	tar -xf "${tmpdl}" -C "${tmpdir}"

	### Success: Move it into place!  And do trailing cleanup.
	mv "${tmpdir}" "${destdir}"
	rm -f "${tmpdl}"
)}
