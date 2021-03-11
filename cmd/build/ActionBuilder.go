package build

import (
	"bufio"
	"fmt"
	"github.com/go-git/go-git/v5"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

/**
Go inbuilt filepath lib doesn't support ** globbing https://github.com/golang/go/issues/11862
So, implemented using walk. If we need to glob on other files later, then will include other Go libs for this
*/
func GlobFiles(rootDir string, pattern regexp.Regexp) []string {
	fileList := []string{}
	filepath.Walk(rootDir, func(path string, f os.FileInfo, err error) error {
		if pattern.MatchString(path) {
			fileList = append(fileList, path)
		}
		return nil
	})
	return fileList
}

func DockerBuildVersion(repoDir string) string {
	//git describe --long --always --dirty --match='v*.*'
	// git describe is not implemented in go-git library, hence using format <branch name>-<short commit hash>
	tag := "latest"
	repo, err := git.PlainOpen(repoDir)
	if err != nil || repo == nil {
		log.Println("[WARN] "+repoDir+" is not a Git repo. Docker images will be tagged as `latest`", err)
		return tag
	}
	ref, err := repo.Head()
	if err != nil {
		log.Println(err)
	}
	branch := ref.Name().Short()
	hash := ref.Hash().String()[0:7]

	if branch == "" || hash == "" {
		log.Println("[WARN] "+repoDir+" is not a Git repo. Docker images will be tagged as `latest`", err)
	} else {
		tag = strings.Join([]string{hash, branch}, "-")
	}
	return tag
}

/*
docker build -t ${IMAGE}:${DOCKER_IMAGE_TAG}  -f ${SCRIPT_DIR}/Dockerfile .
docker tag ${IMAGE}:${DOCKER_IMAGE_TAG} ${DOCKER_IMAGE}
cortex docker login
docker push ${DOCKER_IMAGE}
*/
// Later this will be replaced with daemonless & rootless build
func BuildActionImage(namespace string, name string, version string, dockerfile string, buildContext string, dockerRegistry string) string {
	var dockerImage = fmt.Sprint(namespace, "/", name, ":", version)
	var dockerTag = fmt.Sprint(dockerRegistry, "/", dockerImage)

	var dockerBuildCmd = strings.Join([]string{"docker build -t", dockerImage, "-f", dockerfile, buildContext}, " ")
	log.Println("Building: ", dockerBuildCmd)
	NativeExitOnError(dockerBuildCmd)

	var dockerTagCmd = strings.Join([]string{"docker tag", dockerImage, dockerTag}, " ")
	NativeExitOnError(dockerTagCmd)

	log.Println("Pushing docker image tag: ", dockerTag)
	NativeExitOnError(fmt.Sprint("docker push ", dockerTag))

	return dockerTag
}

func DockerLogin(dockerRegistry string, dockerUser string, dockerPassword string) {
	NativeExitOnError(strings.Join([]string{"docker login", "-u", dockerUser, "--password", dockerPassword, dockerRegistry}, " "))
}

/**
This method executes shell commands and returns command output logs, but os.Exit if program returns exit code > 0. So caller doesn't have to handle error.
We need to create other method to return error and don't exit for scenario where we shouldn't exit app on a command failure
*/
func NativeExitOnError(cmd string) {
	hostOs := runtime.GOOS
	var shell *exec.Cmd
	switch hostOs {
	case "windows":
		shell = exec.Command("cmd", "/C", cmd)
	default:
		shell = exec.Command("/bin/sh", "-c", cmd)
	}

	// stream output of command
	stdout, _ := shell.StdoutPipe()
	shell.Stderr = shell.Stdout
	err := shell.Start()

	scanner := bufio.NewScanner(stdout)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		m := scanner.Text()
		log.Print(m)
	}
	err = shell.Wait()

	if err != nil {
		log.Println("Failed to execute: ", cmd)
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode := exitError.ExitCode()
			if exitCode > 0 {
				log.Fatalln(" Exit Code [", exitCode, "] failed with error ", err)
				os.Exit(exitCode)
			}
		} else {
			log.Fatalln(err)
		}
	}
}
