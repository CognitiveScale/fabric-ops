package cmd

import (
	"archive/zip"
	"bytes"
	"crypto/tls"
	"encoding/pem"
	"errors"
	"fabric-ops/cmd/build"
	"fabric-ops/cmd/deploy"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"github.com/tidwall/gjson"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	defaultManifestFile = "fabric.yaml"
)

var (
	dockerfileRegex, _   = regexp.Compile("Dockerfile")
	jsonYamlFileRegex, _ = regexp.Compile(".*\\.(json|yaml)")
	pathSep, _           = regexp.Compile(`\\|/`)
)

var rootCmd = &cobra.Command{
	Use:                   "fabric <RepoRootDir> [-m <manifest file>]",
	Args:                  validateArgs,
	DisableFlagsInUseLine: true,
	Short:                 "Cortex GitOps CLI for deployment of Cortex resources",
	Long: `This app:
		* Build & push Docker images for Cortex Action
		* Deploy Cortex assets described in manifest file <fabric.yaml>
	`,
	DisableAutoGenTag: true,
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("Building Cortex Actions in repo checkout ", args[0])
		var repoDir = args[0]
		var dockerfiles = build.GlobFiles(repoDir, *dockerfileRegex)
		mapping := map[string]string{} // get docker images built

		if len(dockerfiles) == 0 {
			log.Println("No Dockerfiles found in ", repoDir)
		} else {
			log.Println("Repo ", repoDir, " Dockerfiles ", dockerfiles)
			var gitTag = build.DockerBuildVersion(repoDir)
			var namespace = deploy.GetEnvVar("DOCKER_PREGISTRY_PREFIX")
			dockerimages := buildActionImages(dockerfiles, repoDir, gitTag, namespace)
			for _, image := range dockerimages {
				mapping[deploy.DockerImageName(image)] = image
			}
		}

		manifestFile := cmd.Flag("manifest").Value.String()
		if manifestFile == "" {
			manifestFile = defaultManifestFile
		}
		//deploy
		deployCortexManifest(repoDir, manifestFile, mapping)
	},
}

var buildCmd = &cobra.Command{
	Use:                   "build  <RepoRootDir>",
	Args:                  validateArgs,
	DisableFlagsInUseLine: true,
	Short:                 "Search for Dockerfile(s) in Git repo and builds Docker images",
	Long:                  `Follows convention: Build docker image using Dockerfile and configured build context, <DOCKER_PREGISTRY_PREFIX as namespace>/<image name as parent dir>:g<Git tag and version>, and return build image details`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("Building Cortex Actions in repo checkout ", args[0])
		var repoDir = args[0]
		var dockerfiles = build.GlobFiles(repoDir, *dockerfileRegex)
		if len(dockerfiles) == 0 {
			log.Println("No Dockerfile found in ", repoDir)
			return
		}

		var gitTag = build.DockerBuildVersion(repoDir)
		var namespace = deploy.GetEnvVar("DOCKER_PREGISTRY_PREFIX")

		buildActionImages(dockerfiles, repoDir, gitTag, namespace)
	},
}

var deployCmd = &cobra.Command{
	Use:                   "deploy  <RepoRootDir>  [-m <manifest file>]",
	Args:                  validateArgs,
	DisableFlagsInUseLine: true,
	Short:                 "Deploys Cortex Resources from manifest file <fabric.yaml>",
	Long:                  `Deploys Cortex Resources from manifest file <fabric.yaml>`,
	Run: func(cmd *cobra.Command, args []string) {
		var repoDir = args[0]
		/*
			`actionImageMapping` is actionName (and docker image name) to docker image URL in registry mapping. This is required for substituting
			docker image in action definition exported from one environment and deploying to other environment.

			Currently, invoking this will not perform docker image substitution and action deployment may fail, unless deploying action in same DCI from where
			its exported or image exists in the DCI (may be manually copied or docker registry is shared within multiple DCIs)

			Alternatively, we can query docker registry based on image name. But this will add dependency on registry tools/plugins for search. Better use root cmd for action substitution
		*/
		manifestFile := cmd.Flag("manifest").Value.String()
		if manifestFile == "" {
			manifestFile = defaultManifestFile
		}
		//deploy
		log.Println("Deploying Cortex resources from manifest ", manifestFile, " in repo ", repoDir)
		deployCortexManifest(repoDir, manifestFile, nil)
	},
}

var dockerLoginCmd = &cobra.Command{
	Use: "dockerAuth <DockerRegistryURL> <User> <Password>",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 3 {
			return errors.New("requires 3 args: <DockerRegistryURL> <User> <Password>")
		}
		return nil
	},
	DisableFlagsInUseLine: true,
	Short:                 "Docker login for pushing images",
	Long:                  "Docker login for pushing images",
	Run: func(cmd *cobra.Command, args []string) {
		dockerRegistry := args[0]
		dockerUser := args[1]
		dockerPassword := args[2]

		build.DockerLogin(dockerRegistry, dockerUser, dockerPassword)
		log.Println("Docker login successful")
	},
}

func buildActionImages(dockerfiles []string, repoDir string, gitTag string, namespace string) []string {
	cortex := createCortexClientFromConfig()
	registry := deploy.GetEnvVar("DOCKER_PREGISTRY_URL")
	if namespace == "" {
		namespace = cortex.GetAccount()
	}
	if registry == "" {
		registry = cortex.GetDockerRegistry()
	} else {
		registry = strings.Trim(registry, "/")
	}

	log.Println("Building Docker images with tag: ", gitTag, " and namespace: ", namespace, ". Pushing to registry: ", registry)

	dockerimages := []string{}
	for _, dockerfile := range dockerfiles {
		log.Println("Building ", dockerfile)
		var name = filepath.Base(filepath.Dir(dockerfile))
		dockerimages = append(dockerimages, build.BuildActionImage(namespace, name, gitTag, dockerfile, getBuildContext(repoDir, dockerfile), registry))
	}
	return dockerimages
}

func getBuildContext(repoDir string, dockerfile string) string {
	buildContext := deploy.GetEnvVar("DOCKER_BUILD_CONTEXT")
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

func checkTransformerExists(repoDir string) map[string]bool {
	scriptTypeExists := map[string]bool{}
	resourceTypes := []string{"agent", "snapshot", "skill, action", "connection"}
	for _, resourceType := range resourceTypes {
		scriptPath := filepath.Join(repoDir, ".fabric", "_transformers", resourceType+".jsonnet")
		if _, err := os.Stat(scriptPath); os.IsExist(err) {
			scriptTypeExists[resourceType] = true
		} else {
			scriptTypeExists[resourceType] = false
		}
	}
	return scriptTypeExists
}

func transformResource(resourceType string, repoDir string, relPath string, manifestFilePath string) string {
	scriptPath := filepath.Join(repoDir, ".fabric", "_transformers", resourceType+".jsonnet")
	json, err := deploy.Transform(filepath.Join(repoDir, relPath), scriptPath, resourceType, repoDir, manifestFilePath)
	if err != nil {
		log.Fatalln("Failed to transform resource", relPath, "using", scriptPath, err)
	}
	resourcePath := filepath.Join(repoDir, "_tmp", relPath) + ".json"
	deploy.WriteToPath(resourcePath, []byte(json))
	return resourcePath
}

func deployCortexManifest(repoDir string, manifestFilePath string, actionImageMapping map[string]string) {
	var cortex = createCortexClientFromConfig()

	// check if transformer jsonnet script exists for the resource type
	scriptTypeExists := checkTransformerExists(repoDir)
	// process manifest
	manifest := deploy.NewManifest(filepath.Join(repoDir, manifestFilePath))
	for _, action := range manifest.Cortex.Actions {
		relPath := parseManifestResourcePath(action)
		if scriptTypeExists["action"] {
			transformedResource := transformResource("action", repoDir, relPath, manifestFilePath)
			cortex.DeployAction(transformedResource)
		} else {
			cortex.DeployAction(filepath.Join(repoDir, relPath))
		}
	}
	for _, skill := range manifest.Cortex.Skills {
		relPath := parseManifestResourcePath(skill)
		if scriptTypeExists["skill"] {
			transformedResource := transformResource("skill", repoDir, relPath, manifestFilePath)
			cortex.DeploySkill(transformedResource)
		} else {
			cortex.DeploySkill(filepath.Join(repoDir, relPath))
		}
	}
	for _, agent := range manifest.Cortex.Agents {
		relPath := parseManifestResourcePath(agent)
		if scriptTypeExists["agent"] {
			transformedResource := transformResource("agent", repoDir, relPath, manifestFilePath)
			cortex.DeployAgent(transformedResource)
		} else {
			cortex.DeployAgent(filepath.Join(repoDir, relPath))
		}
	}
	for _, snapshot := range manifest.Cortex.Snapshots {
		relPath := parseManifestResourcePath(snapshot)
		if scriptTypeExists["snapshot"] {
			transformedResource := transformResource("snapshot", repoDir, relPath, manifestFilePath)
			deploy.DeploySnapshot(cortex, transformedResource, actionImageMapping)
		} else {
			deploy.DeploySnapshot(cortex, filepath.Join(repoDir, relPath), actionImageMapping)
		}
	}
	//depsMapping := manifest.Cortex.Dependencies
	// dependency checking is on hold https://cognitivescale.atlassian.net/browse/FAB-2481
	var campaigns []string
	for _, campaign := range manifest.Cortex.Campaign {
		v6Client, ok := cortex.(*deploy.CortexClientV6)
		if ok {
			relPath := parseManifestResourcePath(campaign)
			campaignPathSplits := pathSep.Split(relPath, 3)
			campaignBasepath := filepath.Join(repoDir, campaignPathSplits[0], campaignPathSplits[1])

			//zip campaign
			zipPath := zipDirectory(campaignBasepath)
			err := deploy.DeployCampaign(*v6Client, zipPath, true, true)
			if err != nil {
				log.Println("Campaign "+campaignPathSplits[1]+"deployment failed with: ", err)
			}
			campaigns = append(campaigns, filepath.Join(campaignPathSplits[0], campaignPathSplits[1]))
			os.Remove(zipPath)
		} else {
			log.Fatalln("Configured Cortex URL and token configured are not of v6. Campaigns are supported in v6 onwards.")
		}
	}
	for _, connection := range manifest.Cortex.Connection {
		relPath := parseManifestResourcePath(connection)
		// skip connections deployed in campaign deployment
		skip := false
		for _, campaign := range campaigns {
			if strings.HasPrefix(connection, campaign) {
				skip = true
				break
			}
		}
		if !skip {
			if scriptTypeExists["connection"] {
				transformedResource := transformResource("connection", repoDir, relPath, manifestFilePath)
				cortex.DeployConnection(transformedResource)
			} else {
				cortex.DeployConnection(filepath.Join(repoDir, relPath))
			}
		}
	}

	log.Println("Deployed all artifacts from manifest", manifestFilePath)
	defer os.RemoveAll(filepath.Join(repoDir, "_tmp"))
}

func zipDirectory(basepath string) string {
	archive, err := os.Create(basepath + ".zip")
	if err != nil {
		panic(err)
	}
	defer archive.Close()
	zipWriter := zip.NewWriter(archive)

	filepath.Walk(basepath, func(path string, f os.FileInfo, err error) error {
		if f.IsDir() {
			return nil
		}
		f1, err := os.Open(path)
		if err != nil {
			panic(err)
		}
		defer f1.Close()
		w1, err := zipWriter.Create(strings.TrimPrefix(path, basepath+"/"))
		if err != nil {
			panic(err)
		}
		if _, err := io.Copy(w1, f1); err != nil {
			panic(err)
		}
		return nil
	})
	defer zipWriter.Close()
	return basepath + ".zip"
}

/**
https://cognitivescale.atlassian.net/browse/FAB-284
This is to fix manifest file generated in windows and executed in *nix systems (or vice versa)
We generate paths in manifest file, so it will never have path characters like \ or / in filenames, so its safe to split and join to reconstruct path for host os
*/
func parseManifestResourcePath(relativePath string) string {
	switch os.PathSeparator {
	case '\\':
		return strings.Join(strings.Split(relativePath, "/"), "\\")
	case '/':
		return strings.Join(strings.Split(relativePath, "\\"), "/")
	default:
		return relativePath
	}
}

func createCortexClientFromConfig() deploy.CortexAPI {
	var url = strings.TrimSpace(strings.Trim(deploy.GetEnvVar("CORTEX_URL"), "/"))
	var account = strings.TrimSpace(deploy.GetEnvVar("CORTEX_ACCOUNT"))
	var user = strings.TrimSpace(deploy.GetEnvVar("CORTEX_USER"))
	var password = strings.TrimSpace(deploy.GetEnvVar("CORTEX_PASSWORD"))
	var token = strings.TrimSpace(deploy.GetEnvVar("CORTEX_TOKEN"))
	// V6
	var pat = strings.TrimSpace(deploy.GetEnvVar("CORTEX_ACCESS_TOKEN_PATH"))
	var patJson = strings.TrimSpace(deploy.GetEnvVar("CORTEX_ACCESS_TOKEN_VALUE"))
	var project = strings.TrimSpace(deploy.GetEnvVar("CORTEX_PROJECT"))

	var cortex deploy.CortexAPI
	if pat != "" || patJson != "" {
		cortex = deploy.NewCortexClientPAT(project, pat)
	} else if patJson != "" {
		cortex = deploy.NewCortexClientPATContent(project, []byte(patJson))
	} else if token != "" {
		if url == "" {
			log.Fatalln(" Cortex URL for the Token not provided. Either token or user/password or Personal Access Token json file path need to be provided.")
		}
		cortex = deploy.NewCortexClientExistingToken(url, account, token)
	} else if user != "" && password != "" {
		if url == "" {
			log.Fatalln(" Cortex URL for the user/password not provided. Either token or user/password or Personal Access Token json file path need to be provided.")
		}
		cortex = deploy.NewCortexClient(url, account, user, password)
	} else {
		//fallback to cortex-cli config
		cortex = getCortexCliV6Config(project)
	}
	//must not be nil
	return cortex
}

func getCortexCliV6Config(project string) deploy.CortexAPI {
	home, error := os.UserHomeDir()
	configFilePath := filepath.Join(home, ".cortex/config")

	log.Println("Creating Cortex client from cortex-cli config ", configFilePath)
	if error != nil {
		log.Fatalln("Failed to read cortex-cli config", error)
	}
	bytes, error := ioutil.ReadFile(configFilePath)
	if error != nil {
		log.Fatalln("Failed to parse cortex-cli config", error)
	}
	config := gjson.ParseBytes(bytes)
	version := config.Get("version").String()
	currentProfile := config.Get("currentProfile").String()
	if version == "3" {
		jwk := config.Get("profiles." + currentProfile)
		if project == "" {
			project = jwk.Get("project").String()
		}
		if project == "" {
			log.Fatalln("Cortex project not provided. Either set CORTEX_PROJECT environment variable or set in cortex-cli config profile")
		}
		log.Println("Created cortex client from cortex-cli config", configFilePath, ". Profile:", currentProfile)
		return deploy.NewCortexClientPATContent(project, []byte(jwk.Raw))
	}
	log.Fatalln("cortex-cli config supported only for V6 (JWK token)")
	return nil
}

func validateArgs(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return errors.New("requires Git repo directory")
	} else if _, err := os.Stat(args[0]); os.IsNotExist(err) {
		return err
	}
	return nil
}

func Execute(version string) {
	rootCmd.Version = version
	rootCmd.SetHelpTemplate("\nVersion: " + version + "\n\n" + rootCmd.HelpTemplate())
	if err := rootCmd.Execute(); err != nil {
		log.Fatalln(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.AddCommand(buildCmd, deployCmd, dockerLoginCmd, generateDocsCmd, extractSSLCertCmd)
	rootCmd.Flags().StringP("manifest", "m", defaultManifestFile, "Relative path of Manifest file <fabric.yaml>")
	deployCmd.Flags().StringP("manifest", "m", defaultManifestFile, "Relative path of Manifest file <fabric.yaml>")

	generateDocsCmd.Flags().StringP("format", "f", "md", "Documentation format. Defaults to markdown")
	generateDocsCmd.Flags().StringP("out", "o", "doc", "Documentation output directory. Defaults to doc")
}

func initConfig() {
	// DO NOTHING
	//viper.AutomaticEnv()
}

var generateDocsCmd = &cobra.Command{
	Use:   "docgen  [-f <md>] [-o <./doc>]",
	Short: "Generate documentation for this CLI tool",
	Long:  `Generate documentation for this CLI tool using Cobra doc generator. By default generates in markdown format in doc directory`,
	Run: func(cmd *cobra.Command, args []string) {
		format := cmd.Flag("format").Value.String()
		out := cmd.Flag("out").Value.String()

		err := os.MkdirAll(out, os.FileMode(0755))
		if err != nil {
			log.Println(err) // this will be due to directory already exists
		}
		if format != "md" {
			log.Fatalln("Currently only markdown is supported")
		}
		err = doc.GenMarkdownTree(rootCmd, out)
		if err != nil {
			log.Fatalln(err)
		}
	},
}

var extractSSLCertCmd = &cobra.Command{
	Use:   "fetchCert  <Server URL> <Path to save cert>",
	Short: "Download SSL certificate from server",
	Long:  `Download SSL certificate from server to add as trusted, in case its not from a public CA`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 2 {
			return errors.New("requires Git repo directory")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		conn, err := tls.Dial("tcp", args[0], &tls.Config{
			InsecureSkipVerify: true,
		})
		if err != nil {
			log.Fatalln("Failed to connect to server", err)
		}
		defer conn.Close()
		var b bytes.Buffer
		for _, cert := range conn.ConnectionState().PeerCertificates {
			err := pem.Encode(&b, &pem.Block{
				Type:  "CERTIFICATE",
				Bytes: cert.Raw,
			})
			if err != nil {
				log.Fatalln("Failed to save certificate", err)
			}
		}
		ioutil.WriteFile(args[1], []byte(b.String()), 0400)
	},
}
