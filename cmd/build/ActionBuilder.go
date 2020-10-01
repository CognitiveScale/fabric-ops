package build

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

/**
Go inbuilt filepath lib doesn't support ** globbing https://github.com/golang/go/issues/11862
So, implemented using walk. If we need to glob on other files later, then will include other Go libs for this
*/
func GlobDockerfiles(rootDir string) []string {
	fileList := []string{}
	filepath.Walk(rootDir, func(path string, f os.FileInfo, err error) error {
		if strings.HasSuffix(path, "Dockerfile") {
			fileList = append(fileList, path)
		}
		return nil
	})
	return fileList
}

func DockerBuildVersion(repoDir string) string {
	var gitTagCmd = fmt.Sprint("cd ", repoDir, " && git rev-parse --short HEAD")
	var tag = Native(gitTagCmd)
	return fmt.Sprint("g", strings.TrimSpace(tag))
}

/*
docker build -t ${IMAGE}:${DOCKER_IMAGE_TAG}  -f ${SCRIPT_DIR}/Dockerfile .
docker tag ${IMAGE}:${DOCKER_IMAGE_TAG} ${DOCKER_IMAGE}
cortex docker login
docker push ${DOCKER_IMAGE}
*/
//TODO add command success validation and exception handling
// Later this will be replaced with daemonless & rootless build
func BuildActionImage(namespace string, name string, version string, dockerfile string, buildContext string, dockerRegistry string, cortexToken string) string {
	var dockerImage = fmt.Sprint(namespace, "/", name, ":", version)
	var dockerTag = fmt.Sprint(dockerRegistry, "/", dockerImage)

	var dockerBuildCmd = strings.Join([]string{"docker build -t", dockerImage, "-f", dockerfile, buildContext}, " ")
	fmt.Println(dockerBuildCmd)
	var logs = Native(dockerBuildCmd)
	fmt.Println(logs)

	var dockerTagCmd = strings.Join([]string{"docker tag", dockerImage, dockerTag}, " ")
	logs = Native(dockerTagCmd)
	fmt.Println(logs)

	logs = Native(strings.Join([]string{"docker login", "-u", "cli", "--password", cortexToken, dockerRegistry}, " "))

	logs = Native(fmt.Sprint("docker push ", dockerTag))
	fmt.Println(logs)

	return dockerTag
}

func Native(cmd string) string {
	out, err := exec.Command("/bin/sh", "-c", cmd).Output()
	if err != nil {
		fmt.Println(cmd, " failed with error ", err)
	}
	output := string(out)
	return output
}
