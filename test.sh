#!/usr/bin/env bash
set -e

# for mac its `darwin`. see `go tool dist list`
#export GOOS=linux
#export GOOS=windows
export CORTEX_URL=https://api.anthem-modeloffice-workos.insights.ai
export CORTEX_ACCOUNT=workos-dev
export CORTEX_TOKEN=
export CORTEX_USER=ljha
export CORTEX_PASSWORD=
export DOCKER_PREGISTRY_PREFIX=c12e
export DOCKER_PREGISTRY_URL=private-registry.dev01-cts-aic.insights.ai
export DOCKER_BUILD_CONTEXT=DOCKERFILE_CURRENT_DIR # DOCKERFILE_CURRENT_DIR | DOCKERFILE_PARENT_DIR | REPO_ROOT | </path/relative/to/repo>
#export DOCKER_BUILDKIT=1

#Build cmd tool.
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
    time ./fabric deploy $1 $2 $3
}

function all() {
    ./fabric $1 $2 $3
}

function dockerLogin() {
    ./fabric dockerAuth $1 $2 $3
}

package
#dockerLogin $DOCKER_PREGISTRY_URL 'cli' $CORTEX_TOKEN

# fabric calls build and pass result of build to deploy for image substitution in action definition.
# calling `fabric deploy` without result of build will not perform image substitution and action deployment may fail, unless deploying action in same DCI
# from where its exported or image exists in the DCI (may be manually copied or docker registry is shared within multiple DCIs)

#all "/Users/ljha/xcognitive/cortex-reference-models"

#build "/Users/ljha/xcognitive/cortex-reference-models/src/bank_marketing"

# -m or --manifest to optionally prove manifest file, if there are multiple manifest or with different name
#deploy "/Users/ljha/xcognitive/cortex-reference-models" -m "fabric.yaml.0"
deploy "/Users/ljha/xcognitive/cortex-reference-models"
