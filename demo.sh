#!/bin/bash
set -eo pipefail

if [ -f .gopath/bin/repeatr ]; then PATH=$PWD/.gopath/bin/:$PATH; fi
demodir="demo";

rm -rf "$demodir"
mkdir -p "$demodir" && cd "$demodir" && demodir="$(pwd)"
echo "$demodir"



set -x



repeatr scan --kind=tar



repeatr run -i <(cat <<EOF
{
	"Inputs": [
		{
			"Type": "tar",
			"Location": "/",
			"Hash": "b6nXWuXamKB3TfjdzUSL82Gg1avuvTk0mWQP4wgegscZ_ZzG9GfHDwKXQ9BfCx6v",
			"URI": "assets/ubuntu.tar.gz"
		}
	],
	"Accents": {
		"Entrypoint": [ "echo", "Hello from repeatr!" ]
	},
	"Outputs": [
		{
			"Type": "tar",
			"Location": "/var/log",
			"URI": "basic.tar"
		}
	]
}
EOF
)


