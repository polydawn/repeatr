#!/bin/bash
set -euo pipefail

rm -r .pipeline.demo || true
mkdir .pipeline.demo
cd .pipeline.demo

hitch init
hitch catalog create "demo.polydawn.net/pipeline/base"
hitch release start "demo.polydawn.net/pipeline/base" "v0.1"
hitch release add-item "linux-amd64" "tar:6q7G4hWr283FpTa5Lf8heVqw9t97b5VoMU6AGszuBYAz9EzQdeHVFAou7c4W9vFcQ6"
hitch release commit
hitch show "demo.polydawn.net/pipeline/base"

frm="$(cat <<EOF
	{
		"formula": {
			"inputs": {
				"/": "$(hitch show "demo.polydawn.net/pipeline/base:v0.1:linux-amd64")"
			},
			"action": {
				"exec": ["/bin/echo", "hello world!"]
			}
		},
		"context": {
			"fetchUrls": {
				"/": [
					"file://./fixtures/busybash.tgz"
				]
			}
		}
	}
EOF
)"
repeatr run <(echo "$frm")