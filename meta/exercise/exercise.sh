#!/bin/bash
set -euo pipefail
trap 'echo -e "\E[31mFUCK\E[0m\n"' ERR

## Utils -- output fmting
function prefix { sed 's/^/'"$1"'/' ; }
function indent { prefix '\t' ; }
function msg1 { echo -e "\E[0;35m===" "$@" "===\E[0m" ; }
function msg1a { echo -e "\E[0;35m↓↓↓≡≡" "$@" "≡≡↓↓↓\E[0m" ; }
function msg1b { echo -e "\E[0;35m↑↑↑≡≡" "$@" "≡≡↑↑↑\E[0m" ; }
function msg2 { echo -e "\E[1;34m⇨" "$@" "\E[0m" ; }
function frame { msg1a "$@" ; indent ; msg1b "done ($@)" ; echo ; }
function framedo { "$@" 2>&1 | frame "$@" ; }

## Find ye executable.
if [ -x .gopath/bin/repeatr ]; then PATH=$PWD/.gopath/bin/:$PATH; fi
>&2 echo "Testing with `which repeatr` --"
>&2 echo "`repeatr version | indent`"
>&2 echo

## Choose ye output and working paths.
TMP=${TMP:-"/tmp/repeatr-exercise"}
mkdir -p "$TMP"

## Flaggery.
CI_FLAGS=${CI_FLAGS:-}


## Snips
Snip_Input_Base='
		"/":
			type: "tar"
			hash: "aLMH4qK1EdlPDavdhErOs0BPxqO0i6lUaeRE4DuUmnNMxhHtF56gkoeSulvwWNqT"
			silo: "http+ca://repeatr.s3.amazonaws.com/assets/"
'

## Test: basic hello.
(TEST=test-hello
repeatr run $CI_FLAGS <(cat <<EOF
	inputs:
		$Snip_Input_Base
	action:
		command: ["ls", "-la", "/tmp"]
	outputs:
		"/tmp":
			type: "dir"
			silo: "file://$TMP/$TEST"
EOF
) 2>&1 | frame "$TEST"
)


## Test: hello with output filters.
(TEST=test-hello-output-filters
repeatr run $CI_FLAGS <(cat <<EOF
	inputs:
		$Snip_Input_Base
	action:
		command: ["ls", "-la", "/tmp"]
	outputs:
		"/tmp":
			type: "dir"
			silo: "file://$TMP/$TEST"
			filters:
				- uid keep
EOF
) 2>&1 | frame "$TEST"
)


## Test: hello with non-zero exit.
(TEST=test-hello-exit
repeatr run --ignore-job-exit $CI_FLAGS <(cat <<EOF
	inputs:
		$Snip_Input_Base
	action:
		command: ["bash", "-c", "exit 4"]
	outputs:
		"/tmp":
			type: "dir"
			silo: "file://$TMP/$TEST"
			filters:
				- uid keep
EOF
) 2>&1 | grep -C999 '"hash": "4"' | frame "$TEST"
)
