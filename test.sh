#!/usr/bin/env bash
set -e

export CORTEX_URL=https://api.anthem-modeloffice-workos.insights.ai
export CORTEX_ACCOUNT=workos-dev
export CORTEX_TOKEN=
export CORTEX_USER=ljha
export CORTEX_PASSWORD=
export DOCKER_PREGISTRY_PREFIX=c12edemo
export DOCKER_BUILD_CONTEXT=DOCKERFILE_CURRENT_DIR # DOCKERFILE_CURRENT_DIR | DOCKERFILE_PARENT_DIR | REPO_ROOT | </path/relative/to/repo>
export DOCKER_BUILDKIT=1

#Build cmd tool. TODO build for windows, linux and mac
function package() {
    [ -e fabric ] && rm fabric
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

function all() {
    ./fabric $1
}

#package
all "/Users/ljha/xcognitive/cortex-reference-models/src/bank_marketing"
#build "/Users/ljha/xcognitive/cortex-reference-models/src/bank_marketing"
#deploy "/Users/ljha/xcognitive/cortex-reference-models/src/bank_marketing"
