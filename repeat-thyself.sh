#!/bin/bash
#
# Build repeatr repeatedly with repeatr.
#
# If you don't have repeatr on your path yet, you can
# use `./goad sys` to install it to your local `/usr/bin/`,
# or `./goad install` to update a copy in `./.gopath/bin/`.
#
set -euo pipefail

if [ -x .gopath/bin/repeatr ]; then PATH=$PWD/.gopath/bin/:$PATH; fi
if [ ! -d .git ]; then echo "this script assumes it is run from a local git repo containing repeatr." 1>&2 ; exit 1 ; fi



### Set values for metadata our build injects for debugging purposes.
#  These are used by go-generate (see `cli/go.version.tmpl`);
#  In order to produce consistent outputs, we have to affix them.

# Pick out the current head hash.
# Of course you could use any commit hash you want.
GITCOMMIT=${GITCOMMIT:-$(git rev-parse HEAD)}

# Nil builddate by default.  But if you want to set one, go ahead.
BUILDDATE=${BUILDDATE:-"xxx"}



### Arrange a short bootstrapping script
# This sets just enough variables to make our golang package run.
Script="$(cat <<-'EOF'
	#!/bin/bash
	export GOROOT=/app/go/go
	export PATH=$PATH:/app/go/go/bin
	./goad install
EOF
)"
# escape it as a json string.  if you have jq, use `jq -s -R .` instead.
Script="$(echo "${Script}" | tr -d "\t" | grep -v "^#" | tr -s "\n" ";" | sed "s/\"/\\\\\"/g")"



### Assemble the full formula
# This is mostly just taking our metadata variables above, and injecting them.
Formula="$(cat <<-EOF
{
	"inputs": {
		"/": {
			"type": "tar",
			"hash": "uJRF46th6rYHt0zt_n3fcDuBfGFVPS6lzRZla5hv6iDoh5DVVzxUTMMzENfPoboL",
			"silo": "http+ca://repeatr.s3.amazonaws.com/assets/"
		},
		"/app/go/": {
			"type": "tar",
			"hash": "vbl0TwPjBrjoph65IaWxOy-Yl0MZXtXEDKcxodzY0_-inUDq7rPVTEDvqugYpJAH",
			"silo": "https://storage.googleapis.com/golang/go1.5.linux-amd64.tar.gz"
		},
		"/task/repeatr/": {
			"type": "git",
			"hash": "${GITCOMMIT}",
			"silo": "https://github.com/polydawn/repeatr.git"
		}
	},
	"action": {
		"command": [ "/bin/bash", "-c", "${Script}" ],
		"cwd": "/task/repeatr/",
		"env": {
			"GITCOMMIT": "${GITCOMMIT}",
			"BUILDDATE": "${BUILDDATE}"
		}
	},
	"outputs": {
		"executable": {
			"type": "tar",
			"mount": "/task/repeatr/.gopath/bin/",
			"filters": [
				"uid 10100",
				"gid 10100",
				"mtime @0"
			],
			"silo": "file://repeatr.tar"
		}
	}
}
EOF
)"


### run it!
time repeatr run -i <(echo "${Formula}")
