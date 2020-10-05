#!/usr/bin/env bash

export CORTEX_URL=https://api.dev01-cts-aic.insights.ai
export CORTEX_ACCOUNT=cts-aic-dev
export CORTEX_TOKEN=
export CORTEX_USER=ljha
export CORTEX_PASSWORD=
export DOCKER_PREGISTRY_PREFIX=c12e

#Build cmd tool. TODO build for windows, linux and mac
function package() {
    rm fabric
    go build -ldflags "-s -w" -o fabric main.go
    # upx can better compress binaries, but avoiding coz it GPL license and binary size is not important metric
    # upx -9 -k fabric
}

#Build Cortex Actions
function build() {
    time ./fabric build $1
}

#Deploy Cortex artifacts
function deploy() {
    time ./fabric deploy $1
}

package
build "/Users/ljha/xcognitive/c12e-doc-sim"
deploy "/Users/ljha/xcognitive/c12e-doc-sim"
