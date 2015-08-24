#!/bin/bash
set -eo pipefail

if [ -x .gopath/bin/repeatr ]; then PATH=$PWD/.gopath/bin/:$PATH; fi
if [ -x repeatr ]; then PATH=$PWD:$PATH; fi
if [ "$1" != "-t" ]; then straight=true; fi; export straight;
demodir="demo";

cnbrown="$(echo -ne "\E[0;33m")" # prompt
clblack="$(echo -ne "\E[1;30m")" # aside
clbrown="$(echo -ne "\E[1;33m")" # command
clblue="$(echo -ne "\E[1;34m")"  # section docs
cnone="$(echo -ne "\E[0m")"
awaitack() {
	[ "$straight" != true ] && return;
	echo -ne "${cnbrown}waiting for ye to hit enter, afore the voyage heave up anchor and make headway${cnone}"
	read -es && echo -ne "\E[F\E[2K\r"
}
tellRunning() {
	echo -e "${clblack}# running \`${clbrown}$@${clblack}\` >>>${cnone}"
}

rm -rf "$demodir"
mkdir -p "$demodir" && cd "$demodir" && demodir="$(pwd)"
echo "$demodir"
echo "$(which repeatr)"

export REPEATR_BASE="$demodir/repeatr_base"



echo -e "${clblue}# Repeatr says hello!${cnone}"
echo -e "${clblue}#  Without a command, it provides help.${cnone}"
(
	tellRunning "repeatr"
	time repeatr
)
echo -e "${clblue} ----------------------------${cnone}\n\n"
awaitack


echo -e "${clblue}# To suck in data, use the scan command:${cnone}"
echo
(
	tellRunning "repeatr scan --help"
	time repeatr scan --help
	tellRunning "repeatr scan --kind=tar"
	time repeatr scan --kind=tar
)
echo
echo -e "${clblue}#  This determines the data identity,${cnone}"
echo -e "${clblue}#   Uploads it to a warehouse,${cnone}"
echo -e "${clblue}#    And outputs the config to request it again later.${cnone}"
echo -e "${clblue} ----------------------------${cnone}\n\n"
awaitack




echo -e "${clblue}# The \`repeatr run\` command takes a job description and executes it.${cnone}"
echo -e "${clblue}#  Stdout goes to your terminal; any 'output' specifications are saved/uploaded.${cnone}"
echo -e "${clblue}#  This first run might take a while -- it's downloading an operating system image first!${cnone}"
(
	mkdir -p "${demodir}/local-warehouse"
	tellRunning "repeatr run -i some-json-config-files.conf"
	time repeatr run -i <(cat <<-EOF
	{
		"inputs": [
			{
				"type": "tar",
				"mount": "/",
				"hash": "uJRF46th6rYHt0zt_n3fcDuBfGFVPS6lzRZla5hv6iDoh5DVVzxUTMMzENfPoboL",
				"silo": ["http+ca://repeatr.s3.amazonaws.com/assets/"]
			}
		],
		"action": {
			"command": [ "echo", "Hello from repeatr!" ]
		},
		"outputs": [
			{
				"type": "tar",
				"mount": "/var/log",
				"silo": ["file+ca://${demodir}/local-warehouse"]
			}
		]
	}
EOF
	)
)
echo -e "${clblue} ----------------------------${cnone}\n\n"
awaitack




echo -e "${clblue}# The \`repeatr run\` command can use cached assets to start jobs faster.${cnone}"
echo -e "${clblue}#  Here we use the same rootfs image of ubuntu, so it starts instantly.${cnone}"
(
	tellRunning "time repeatr run -i some-json-config-files.conf"
	time repeatr run -i <(cat <<-EOF
	{
		"inputs": [
			{
				"type": "tar",
				"mount": "/",
				"hash": "uJRF46th6rYHt0zt_n3fcDuBfGFVPS6lzRZla5hv6iDoh5DVVzxUTMMzENfPoboL",
				"silo": ["http+ca://repeatr.s3.amazonaws.com/assets/"]
			}
		],
		"action": {
			"command": [ "echo", "Hello from repeatr!" ]
		},
		"outputs": [
			{
				"type": "tar",
				"mount": "/var/log",
				"silo": ["file+ca://${demodir}/local-warehouse"]
			}
		]
	}
EOF
	)
)
echo -e "${clblue}# Also, note that the output is the same hash?${cnone}"
echo -e "${clblue}#  Given the same inputs, this command produces the same outputs, every time. ;)${cnone}"
echo -e "${clblue} ----------------------------${cnone}\n\n"
awaitack



echo -e "${clblue}# We also included an output spec in those last two commands.${cnone}"
echo -e "${clblue}# That means repeatr uploaded our data to the 'warehouse' we gave it --${cnone}"
echo -e "${clblue}#  warehouses act like permanent storage for your data.${cnone}"
echo -e "${clblue}# We just used a local filesystem, so let's see how that looks:${cnone}"
(
	tellRunning "ls -lah \$demodir/local-warehouse"
	ls -lah "${demodir}/local-warehouse"
)
echo -e "${clblue}# Content addressible storage means the same data gets de-duplicated automatically --${cnone}"
echo -e "${clblue}#  since both of our jobs produced the same output, it's just here stored once.${cnone}"
echo -e "${clblue}# Everything in a warehouse has guaranteed integrity based on the hashes.${cnone}"
echo -e "${clblue}#  If a warehouse suffers disk corruption?  You're covered -- you'll know immediately.${cnone}"
echo -e "${clblue}#  If a warehouse gets hacked?  You're covered -- if the attacker changed anything, you'll know immediately!${cnone}"
echo -e "${clblue}# Here we used a local filesystem, but other warehousing options include S3, for example.${cnone}"
echo -e "${clblue} ----------------------------${cnone}\n\n"
awaitack



echo "${clblue}#  That's all!  Neat, eh?${cnone}"


