#GitOps Tool for Fabric

### Inputs:
* Git repo checkout folder with manifest file fabric.yaml and .fabric folder containing Cortex artifacts at root 
* This implementation depends on `git` CLI and `Docker` daemon

### Configurations:
* DOCKER_PREGISTRY_PREFIX: env var for Docker image namespace for all action images in this repo

##### Steps:
- Get Git tag 
- Search Dockerfile(s)
- Build docker image using Dockerfile and repo root as build context, <DOCKER_PREGISTRY_PREFIX as namespace>/ < image name as parent dir >:g< Git tag and version >, and return build image details
- Cortex Client for: cortex docker login, post action, skill, agent and existence check
- Cortex docker login
- Tag and push image
- Deploy Cortex resources in manifest fabric.yaml

https://docs.google.com/document/d/13bP7agrn3RpcWMutc5WbpV_Cusalg2ejTpx17ecudFY/edit#heading=h.q117vxrv0r3w


This is implemented as a CLI app using https://github.com/spf13/cobra
`main.go` is entrypoint
`go.mod` to manage dependency
All commands are in `cmd/root.go`

##### Commands:
* Build & push Cortex Actions Docker images 
`fabric build <Git repo directory>`
* Deploy Cortex resources as per manifest
`fabric deploy <Git repo directory> `
