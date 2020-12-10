package build

import (
	"fmt"
	"github.com/go-git/go-git/v5"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
	//git describe --long --always --dirty --match='v*.*'
	// git describe is not implemented in go-git library, hence using format <branch name>-<short commit hash>
	repo, err := git.PlainOpen(repoDir)
	if err != nil {
		log.Fatalln("Failed to get commit hash. ", repoDir, " must be root of Git repo checkout. Error: ", err)
	}
	ref, err := repo.Head()
	if err != nil {
		log.Fatalln(err)
	}
	branch := ref.Name().Short()
	hash := ref.Hash().String()[0:7]
	if branch == "" || hash == "" {
		log.Fatalln("Failed to fetch branch and HEAD from repo: "+repoDir, err)
		os.Exit(1)
	}
	tag := strings.Join([]string{hash, branch}, "-")
	return tag
}

/*
docker build -t ${IMAGE}:${DOCKER_IMAGE_TAG}  -f ${SCRIPT_DIR}/Dockerfile .
docker tag ${IMAGE}:${DOCKER_IMAGE_TAG} ${DOCKER_IMAGE}
cortex docker login
docker push ${DOCKER_IMAGE}
*/
//TODO add command success validation and exception handling
// Later this will be replaced with daemonless & rootless build
func BuildActionImage(namespace string, name string, version string, dockerfile string, buildContext string, dockerRegistry string) string {
	var dockerImage = fmt.Sprint(namespace, "/", name, ":", version)
	var dockerTag = fmt.Sprint(dockerRegistry, "/", dockerImage)

	var dockerBuildCmd = strings.Join([]string{"docker build -t", dockerImage, "-f", dockerfile, buildContext}, " ")
	log.Println("Building: ", dockerBuildCmd)
	var logs = NativeExitOnError(dockerBuildCmd)
	log.Println(logs)

	var dockerTagCmd = strings.Join([]string{"docker tag", dockerImage, dockerTag}, " ")
	logs = NativeExitOnError(dockerTagCmd)

	log.Println("Pushing docker image tag: ", dockerTag)
	logs = NativeExitOnError(fmt.Sprint("docker push ", dockerTag))
	log.Println(logs)

	return dockerTag
}

func DockerLogin(dockerRegistry string, dockerUser string, dockerPassword string) {
	logs := NativeExitOnError(strings.Join([]string{"docker login", "-u", dockerUser, "--password", dockerPassword, dockerRegistry}, " "))
	log.Println(logs)
}

/**
This method executes shell commands and returns command output logs, but os.Exit if program returns exit code > 0. So caller doesn't have to handle error.
We need to create other method to return error and don't exit for scenario where we shouldn't exit app on a command failure
*/
func NativeExitOnError(cmd string) string {
	hostOs := runtime.GOOS
	var shell *exec.Cmd
	switch hostOs {
	case "windows":
		shell = exec.Command("cmd", "/C", cmd)
	default:
		shell = exec.Command("/bin/sh", "-c", cmd)
	}
	out, err := shell.Output()
	if err != nil {
		log.Println(cmd)
		log.Println(out)
		log.Fatalln(err)
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
