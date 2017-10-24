#!/bin/bash
source goprj.preamble.shlib

## Initialize submodules.
git submodule update --init

## Make sure the self-symlink exists.
##  Should be committed anyway (but it's harmless to repeat,
##  and this is also useful for project-first-steps.)
mkdir -p "$(dirname "$GOPATH/src/$GOPRJ_PKG")"
ln -snf "$(echo "${GOPRJ_PKG//[^\/]}/" | sed s#/#../#g)"../ "$GOPATH/src/$GOPRJ_PKG"
