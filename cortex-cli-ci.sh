#!/usr/bin/env bash
# This scripts is to demonstrate usage of cortex-cli in CI/CD environment. Make necessary changes before deployment
# Prerequisites:
# 	* Install cortex-cli (from https://www.npmjs.com/package/cortex-cli) and jq
# 	* Login to Cortex Console and download `cortex-token.json` from Settings

set -eux

export PROFILE=${CORTEX_PROFILE:-default} # Cortex config name
export PROJECT=${CORTEX_PROJECT:-default}

# This will setup Cortex profile from `cortex-token.json`, and need to be done only once
setup() {
	cortex configure --file $1 --profile ${PROFILE} --project ${PROJECT}
}

# This creates the project and grants provided users READ access on all resources in the project
setup_project() {
	cortex projects save --name ${PROJECT}
	for user in "$@"
	do
		cortex users grant ${user} --project ${PROJECT} --resource ‘*’ --actions READ
	done
}

# Export Cortex Agent flow:
#	Create Snapshot of Agent(s)
#	Export all Agent(s) snapshots

# Snapshot an Agent
snapshot_agent() {
	cortex agents create-snapshot --agentName $1 --title $2 --project ${PROJECT}
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

# Usage example
setup cortex-token.json # needed only once on host

setup_project <user1> <user2>

export_agents <agent1> <agent2>


