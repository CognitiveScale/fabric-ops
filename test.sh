#!/usr/bin/env bash

# either CORTEX_TOKEN or CORTEX_USER + CORTEX_PASSWORD are required
export CORTEX_URL=
export CORTEX_ACCOUNT=
export CORTEX_TOKEN=
export CORTEX_USER=ljha
export CORTEX_PASSWORD=
export DOCKER_PREGISTRY_PREFIX=c12e

go run main.go build "<Git repo checkout directory>"
go run main.go deploy "<Git repo checkout directory>"
