#!/bin/bash
source goprj.preamble.shlib

go install -ldflags "$GOPRJ_LDFLAGS" ./cmd/* && {
	echo -e "\E[1;32minstall successful.\E[0;m\n"
} || {
	echo -e "\E[1;41minstall failed!\E[0;m"
	exit 8
}
