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



# Pick out the current head hash.
# Of course you could use any commit hash you want.
Commit=${Commit:-$(git rev-parse HEAD)}

Script="$(cat <<-'EOF'
	#!/bin/bash
	set -euo pipefail
	set -x
	
	export GOROOT=/app/go/go
	export PATH=$PATH:/app/go/go/bin
	
	# Hack around our own bad metadata insertion.
	#  These are used by go-generate (see `cli/go.version.tmpl`);
	#  In order to produce consistent outputs, we have to affix them.
	export GITCOMMIT="xxx"
	export BUILDDATE="xxx"

	./goad install
EOF
)"
Script="$(echo "${Script}" | tr -d "\t" | grep -v "^#" | tr -s "\n" ";" | sed "s/\"/\\\\\"/g")"
Formula="$(cat <<-EOF
{
	"Inputs": [{
		"Type": "tar",
		"Location": "/",
		"Hash": "uJRF46th6rYHt0zt_n3fcDuBfGFVPS6lzRZla5hv6iDoh5DVVzxUTMMzENfPoboL",
		"URI": "http+ca://repeatr.s3.amazonaws.com/assets/"
	},{
		"Type": "tar",
		"Location": "/app/go/",
		"Hash": "mfUMdLmuysVlW1jEARtm_YKc5PkLxP2Tj-xwEXqEThUGVAWyyCHJyhFXe7OQSgKs",
		"URI": "https://storage.googleapis.com/golang/go1.4.2.linux-amd64.tar.gz"
	},{
		"Type": "git",
		"Location": "/task/repeatr/",
		"Hash": "${Commit}",
		"URI": "https://github.com/polydawn/repeatr.git"
	}],
	"Accents": {
		"Entrypoint": [ "/bin/bash", "-c", "${Script}" ],
		"Cwd": "/task/repeatr/"
	},
	"Outputs": [{
		"Type": "tar",
		"Location": "/task/repeatr/.gopath/bin/"
	}]
}
EOF
)"
time repeatr run -i <(echo "${Formula}")
