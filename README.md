# GitOps Tool for Fabric
This tool is to support deploying Cortex assets in an automated CI/CD pipeline.

### Inputs:
* Git repo checkout folder with manifest file fabric.yaml and .fabric folder containing Cortex artifacts at root 
* This implementation depends on `git` CLI and `Docker` daemon
* Environment variables (this need to be sourced from Vault or other secret store). See test.sh for usage example:
    *  `CORTEX_URL` DCI API URL
    *  `CORTEX_ACCOUNT`
    *  `CORTEX_TOKEN` Either token or user+password is required
    *  `CORTEX_USER`
    *  `CORTEX_PASSWORD`
    *  `DOCKER_PREGISTRY_PREFIX` Docker image namespace. This will be same for all actions in theGit repo.
    *  `DOCKER_PREGISTRY_URL` Docker private registry URL
    *  `DOCKER_BUILD_CONTEXT`  `DOCKERFILE_CURRENT_DIR | DOCKERFILE_PARENT_DIR | REPO_ROOT | </path/relative/to/repo>`
    
### Cortex Assets CI/CD flow
##### Authoring (using `cortex-cli`)
* Deploy all assets and compose Agent(s)
* Snapshot Agent(s)
* Export all snapshots(s) with manifest file (`cortex deploy snapshots`)
* Push exported snapshots `.fabric` directory and `fabric.yaml` manifest file to Git repo (root level)

##### Deploying (`fabric` this tool)
Set environment variables and run `fabric <Git repo directory>` to deploy all Cortex assets exported in previous Authoring step. This command will:
* Scan Git repo directory recursively for Dockerfile(s)
* Build & tag docker images with configured build context, namespace and git version
* Push built Docker image to configured docker registry

* Parse manifest `fabric.yaml` to get cortex artifacts to be deployed
* Deploy agent, skill, action, datasets and agent snapshots. connection and variables will be in next iteration, because we need to handle secrets
* Action deployment need to substitute image with newly build (namespace, registry url, version change etc). for substitution action name must be docker image name.

> The action name and the docker image name a to be directory name of dockerfile. This is the only convention need to be followed in Git repo.

### Implementation

This is implemented as a CLI app using https://github.com/spf13/cobra
`main.go` is entrypoint
`go.mod` to manage dependency
All commands are in `cmd/root.go`

##### Commands:
*Note: calling `fabric deploy` without the output of `build` (when fabric build and fabric deploy are used separately) will not perform image substitution and action deployment may fail, unless deploying action in same DCI from where its exported or image exists in the DCI (may be manually copied or docker registry is shared within multiple DCIs)*
* Run end-to-end flow (build + deploy)
For end-to-end deployment use `fabric` command as follows
>  `fabric <Git repo directory>`
* Build & push Cortex Actions Docker images
If user is managing docker images in the registries manually, they can use the following command to build or they can build it just using docker.Eg:Build once and coping to multiple registries or copy to same registry in different namespaces
>  `fabric build <Git repo directory>`
* When trying to deploy to Cortex, we need to login to the private registry using the following command
> `fabric dockerAuth $DOCKER_PREGISTRY_URL 'cli' $CORTEX_TOKEN`
* Deploy Cortex resources as per manifest
>  `fabric deploy <Git repo directory> `
