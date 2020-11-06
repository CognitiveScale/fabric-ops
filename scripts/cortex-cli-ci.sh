#!/usr/bin/env bash
# This scripts is to demonstrate usage of cortex-cli in CI/CD environment. Make necessary changes before deployment
# Prerequisites:
# 	* Install `cortex-cli` (from https://www.npmjs.com/package/cortex-cli) and jq
#   * Install `fabric` (download & install latest released binary from https://github.com/CognitiveScale/fabric-ops/releases)
# 	* Login to Cortex Console and download `cortex-token.json` from Settings
#
# See CI/CD flow in README.md (authoring & export of Cortex assets are done using `cortex-cli` and deployment of exported assets are done using `fabric`

set -eux

export CORTEX_PROJECT=<Cortex project>
export CORTEX_ACCESS_TOKEN_PATH=<Personal Access Token `cortex-token.json` downloaded from Settings>

# `cortex-cli` env vars
export PROFILE=${CORTEX_PROFILE:-default} # Cortex config name

# `fabric` env vars
export DOCKER_PREGISTRY_URL=<Docker registry URL to push Cortex Action images>
export DOCKER_PREGISTRY_PREFIX=<Docker registry namespace for Cortex Action images>
export DOCKER_BUILD_CONTEXT=DOCKERFILE_CURRENT_DIR # DOCKERFILE_CURRENT_DIR | DOCKERFILE_PARENT_DIR | REPO_ROOT | </path/relative/to/repo>

# This will setup Cortex profile from `cortex-token.json`, and need to be done only once
setup() {
	cortex configure --file ${CORTEX_ACCESS_TOKEN_PATH} --profile ${PROFILE} --project ${CORTEX_PROJECT}
}

# This creates the project and grants provided users READ access on all resources in the project
setup_project() {
	cortex projects save --name ${CORTEX_PROJECT}
	for user in "$@"
	do
		cortex users grant ${user} --project ${CORTEX_PROJECT} --resource ‘*’ --actions READ
	done
}

# Export Cortex Agent flow:
#   Authoring (cortex-cli) steps
#	    Create Snapshot of Agent(s)
#	    Export all Agent(s) snapshots
#       Push snapshots to git repo
#   Deploying exported assets in Cortex DCI
#       Run `fabric <git repo checkout>`

## cortex-cli usage examples

# Snapshot an Agent
snapshot_agent() {
	cortex agents create-snapshot --agentName $1 --title $2 --project ${CORTEX_PROJECT}
}

# Export all snapshots. -f option overwrite existing exports, if any. This is suitable in CI/CD environment.
export_snapshots() {
	cortex deploy snapshots "$1" -y -f
}

# This function accepts space separated agent names and snapshot each of them, then exports all snapshots
export_agents() {
	snapshot_ids=""

	for agent in "$@"
	do
		snapshot_id=$(snapshot_agent ${agent} "DeploymentSnapshot" | jq -r -j .snapshotId)
		snapshot_ids="${snapshot_ids} ${snapshot_id}"
	done

	snapshot_ids=`echo $snapshot_ids | sed 's/ *$//g'`
	export_snapshots "${snapshot_ids}"
}

## Usage examples. Authoring and deployment are two separate phase in development lifecycle.
# Authoring generates artifacts for deployment & done on developer machine (using cortex-cli) and deployment on CI/CD server (using fabric)

# authoring

setup # auth setup, needed only once on host

setup_project <user1> <user2> # create project and add users (if not already done)

export_agents <agent1> <agent2> # snapshot & export agents

# deployment

#Build Cortex Actions
function build() {
    fabric build $1
}

#Deploy Cortex artifacts
function deploy() {
    fabric deploy $1
}

function buildDeploy() {
    fabric $1
}

function dockerLogin() {
    fabric dockerAuth $1 $2 $3
}

dockerLogin $DOCKER_PREGISTRY_URL <Docker Registry User> <Docker Registry Password> # required once on host to push Docker images

buildDeploy "<Git repo checkout with exported snapshots (`.fabric` directory with Cortex assets and `fabric.yaml` manifest file for driving deployment>"
