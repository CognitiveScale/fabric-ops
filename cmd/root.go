package cmd

import (
	"errors"
	"fabric-ops/cmd/build"
	"fabric-ops/cmd/deploy"
	"fmt"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

var cfgFile = "resources/fabric-defaults.yml"

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "fabric",
	Args:  validateArgs,
	Short: "Cortex GitOps CLI for deployment of Cortex resources",
	Long: `This app:
		* Build & push Docker images for Cortex Action
		* Deploy Cortex assets described in manifest fabric.yaml
	`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("Building Cortex Actions in repo checkout ", args[0])
		var repoDir = args[0]
		var dockerfiles = build.GlobDockerfiles(repoDir)

		var gitTag = build.DockerBuildVersion(repoDir)
		var namespace = viper.GetString("DOCKER_PREGISTRY_PREFIX")

		dockerimages := buildActionImages(dockerfiles, repoDir, gitTag, namespace)
		mapping := map[string]string{}
		for _, image := range dockerimages {
			mapping[deploy.DockerImageName(image)] = image
		}

		//deploy
		deployCortexManifest(repoDir, mapping)
	},
}

var buildCmd = &cobra.Command{
	Use:   "build",
	Args:  validateArgs,
	Short: "Search for Dockerfile(s) in Git repo and builds Docker images",
	Long:  `Follows convention: Build docker image using Dockerfile and repo root as build context, <DOCKER_PREGISTRY_PREFIX as namespace>/<image name as parent dir>:g<Git tag and version>, and return build image details`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("Building Cortex Actions in repo checkout ", args[0])
		var repoDir = args[0]
		var dockerfiles = build.GlobDockerfiles(repoDir)

		var gitTag = build.DockerBuildVersion(repoDir)
		var namespace = viper.GetString("DOCKER_PREGISTRY_PREFIX")

		buildActionImages(dockerfiles, repoDir, gitTag, namespace)
	},
}

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Args:  validateArgs,
	Short: "Deploys Cortex Resources from manifest fabric.yaml",
	Long:  `Deploys Cortex Resources from manifest fabric.yaml`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Deploying Cortex resources from manifest fabric.yaml")
		var repoDir = args[0]

		deployCortexManifest(repoDir, nil)
	},
}

func buildActionImages(dockerfiles []string, repoDir string, gitTag string, namespace string) []string {
	cortex := createCortexClientFromConfig()
	registry := viper.GetString("DOCKER_PREGISTRY_URL")
	if registry == "" {
		registry = cortex.GetDockerRegistry()
	}

	log.Println("Building with tag: ", gitTag, " and namespace: ", namespace, ". Pushing to registry: ", registry)

	build.DockerLogin(registry, cortex.Token)

	dockerimages := []string{}
	for _, dockerfile := range dockerfiles {
		log.Println("Building ", dockerfile)
		var name = path.Base(path.Dir(dockerfile))
		dockerimages = append(dockerimages, build.BuildActionImage(namespace, name, gitTag, dockerfile, getBuildContext(repoDir, dockerfile), registry))
	}
	return dockerimages
}

func getBuildContext(repoDir string, dockerfile string) string {
	buildContext := viper.GetString("DOCKER_BUILD_CONTEXT")
	switch buildContext {
	case "", "DOCKERFILE_CURRENT_DIR":
		return filepath.Dir(dockerfile)
	case "DOCKERFILE_PARENT_DIR":
		return filepath.Dir(filepath.Dir(dockerfile))
	case "REPO_ROOT":
		return repoDir
	default:
		return buildContext
	}
}

func deployCortexManifest(repoDir string, actionImageMapping map[string]string) {
	var cortex = createCortexClientFromConfig()

	//TODO add validation
	manifest := deploy.NewManifest(repoDir + "/fabric.yaml")
	for _, action := range manifest.Actions {
		cortex.DeployAction(repoDir + "/" + action)
	}
	for _, skill := range manifest.Skills {
		cortex.DeployAction(repoDir + "/" + skill)
	}
	for _, agent := range manifest.Agents {
		cortex.DeployAction(repoDir + "/" + agent)
	}
	for _, snapshot := range manifest.Snapshots {
		cortex.DeploySnapshot(repoDir+"/"+snapshot, actionImageMapping)
	}
}

func createCortexClientFromConfig() deploy.CortexClient {
	var url = viper.GetString("CORTEX_URL")
	var account = viper.GetString("CORTEX_ACCOUNT")
	var user = viper.GetString("CORTEX_USER")
	var password = viper.GetString("CORTEX_PASSWORD")
	var token = viper.GetString("CORTEX_TOKEN")

	var cortex deploy.CortexClient
	if len(strings.TrimSpace(token)) > 0 {
		cortex = deploy.NewCortexClientExistingToken(url, account, token)
	} else {
		cortex = deploy.NewCortexClient(url, account, user, password)
	}
	return cortex
}

func validateArgs(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return errors.New("requires Git repo directory")
	} else if _, err := os.Stat(args[0]); os.IsNotExist(err) {
		return err
	}
	return nil
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.fabric.yaml)")

	rootCmd.AddCommand(buildCmd, deployCmd)
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".fabric" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".fabric")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
