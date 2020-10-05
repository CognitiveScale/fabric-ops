package build

import (
	"fmt"
	"log"
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
	var tag = NativeExitOnError(gitTagCmd)
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
	log.Println("Building: ", dockerBuildCmd)
	var logs = NativeExitOnError(dockerBuildCmd)
	log.Println(logs)

	var dockerTagCmd = strings.Join([]string{"docker tag", dockerImage, dockerTag}, " ")
	logs = NativeExitOnError(dockerTagCmd)
	log.Println(logs)

	logs = NativeExitOnError(strings.Join([]string{"docker login", "-u", "cli", "--password", cortexToken, dockerRegistry}, " "))

	logs = NativeExitOnError(fmt.Sprint("docker push ", dockerTag))
	log.Println(logs)

	return dockerTag
}

/**
This method executes shell commands and returns command output logs, but os.Exit if program returns exit code > 0. So caller doesn't have to handle error.
We need to create other method to return error and don't exit for scenario where we shouldn't exit app on a command failure
*/
func NativeExitOnError(cmd string) string {
	out, err := exec.Command("/bin/sh", "-c", cmd).Output()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode := exitError.ExitCode()
			if exitCode > 0 {
				log.Fatalln(cmd, " Exit Code [", exitCode, "] failed with error ", err)
				os.Exit(exitCode)
			}
		}
	}
	output := string(out)
	return output
}
