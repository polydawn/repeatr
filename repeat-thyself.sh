#!/bin/bash
#
# Build repeatr repeatedly with repeatr.
#
# If you don't have repeatr on your path yet, you can
# use `./goad sys` to install it to your local /usr/bin.
#
set -euo pipefail


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
Script="$(echo "${Script}" | jq -s -R .)"
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
		"Hash": "HEAD",
		"URI": "./"
	}],
	"Accents": {
		"Entrypoint": [ "/bin/bash", "-c", ${Script} ],
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
