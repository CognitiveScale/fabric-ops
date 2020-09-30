package main

import (
	"fabric-ops/cmd/build"
	"fabric-ops/cmd/deploy"
	"fmt"
	"path"
)

//cortex
var url = "https://api.dev01-cts-aic.insights.ai"
var account = "cts-aic-dev"

//git
var repoDir = "/Users/ljha/xcognitive/c12e-doc-sim"

//TODO read above config from cobra CLI params/config

func main() {
	var cortex = deploy.NewCortexClient(url, account, "ljha", "")
	fmt.Println(cortex)

	var dockerfiles = build.GlobDockerfiles(repoDir)
	fmt.Println(path.Base(path.Dir(dockerfiles[0])))

	var gitTag = build.DockerBuildVersion(repoDir)
	fmt.Println(gitTag)
}
