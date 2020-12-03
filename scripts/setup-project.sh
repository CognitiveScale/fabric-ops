#!/usr/bin/env bash
# This scripts is to demonstrate usage of cortex-cli to setup new project. Make necessary changes before deployment
# Prerequisites:
# 	* Install `cortex-cli` (from https://www.npmjs.com/package/cortex-cli) and jq
# 	* Login to Cortex Console and download `cortex-token.json` from Settings

set -eux

# Update this to cortex token path
export CORTEX_ACCESS_TOKEN_PATH=cortex-token.json

# `cortex-cli` env vars
export PROFILE=${CORTEX_PROFILE:-default} # Cortex config name

# This will setup Cortex profile from `cortex-token.json`, and need to be done only once
setup() {
	cortex configure --file ${CORTEX_ACCESS_TOKEN_PATH} --profile ${PROFILE} --project ${CORTEX_PROJECT}
}

# create a project and set roles & grants
create_cortex_project() {
    cortex projects save --name $1 --description $2 --title $3

    cortex roles project $1 --roles data-engineer ai-developer business-user

    cortex roles grant ai-developer --project $1 --actions '*' --resource agents
    cortex roles grant ai-developer --project $1 --actions '*' --resource campaigns
    cortex roles grant ai-developer --project $1 --actions READ --resource connections
    cortex roles grant ai-developer --project $1 --actions READ --resource secrets

    cortex roles grant data-engineer --project $1 --actions '*' --resource datasources
    cortex roles grant data-engineer --project $1 --actions '*' --resource profiles
    cortex roles grant data-engineer --project $1 --actions '*' --resource connections
    cortex roles grant data-engineer --project $1 --actions '*' --resource secrets

    cortex roles grant business-user --project $1 --actions '*' --resource campaigns
}

# keep adding users
grant_roles_to_users() {
    cortex roles assign $2 --users $1
}

# usage examples
project_name="example-project"
project_description="example-project-description"
project_title="example-project-title"

create_cortex_project $project_name $project_description $project_title

grant_roles_to_users <user name> <role name>
