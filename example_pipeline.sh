#!/bin/bash
set -euo pipefail

rm -r .pipeline.demo || true
mkdir .pipeline.demo
cd .pipeline.demo

mkdir wares
mkdir -p wares/6q7/G4hW
ln -s ../../../../fixtures/busybash.tgz wares/6q7/G4hW/6q7G4hWr283FpTa5Lf8heVqw9t97b5VoMU6AGszuBYAz9EzQdeHVFAou7c4W9vFcQ6

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
			},
			"outputs": {
				"/task/out": {"packtype": "tar"}
			}
		},
		"context": {
			"fetchUrls": {"/": ["ca+file://./wares/"]},
			"saveUrls": {"/task/out": "ca+file://./wares/"}
		}
	}
EOF
)"
repeatr run <(echo "$frm")
