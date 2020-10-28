fabric
------

Cortex GitOps CLI for deployment of Cortex resources

### Synopsis

This app: \* Build & push Docker images for Cortex Action \* Deploy
Cortex assets described in manifest file &lt;fabric.yaml&gt;

    fabric <RepoRootDir> [-m <manifest file>]

### Options

      -h, --help              help for fabric
      -m, --manifest string   Relative path of Manifest file <fabric.yaml> (default "fabric.yaml")

fabric build
------------

Search for Dockerfile(s) in Git repo and builds Docker images

### Synopsis

Follows convention: Build docker image using Dockerfile and configured
build context,
<DOCKER_PREGISTRY_PREFIX as namespace>/<image name as parent dir>:g<Git tag and version>,
and return build image details

    fabric build  <RepoRootDir>

### Options

      -h, --help   help for build

fabric deploy
-------------

Deploys Cortex Resources from manifest file &lt;fabric.yaml&gt;

### Synopsis

Deploys Cortex Resources from manifest file &lt;fabric.yaml&gt;

    fabric deploy  <RepoRootDir>  [-m <manifest file>]

### Options

      -h, --help              help for deploy
      -m, --manifest string   Relative path of Manifest file <fabric.yaml> (default "fabric.yaml")

fabric docgen
-------------

Generate documentation for this CLI tool

### Synopsis

Generate documentation for this CLI tool using Cobra doc generator. By
default generates in markdown format in doc directory

    fabric docgen  [-f <md>] [-o <./doc>] [flags]

### Options

      -f, --format string   Documentation format. Defaults to markdown (default "md")
      -h, --help            help for docgen
      -o, --out string      Documentation output directory. Defaults to doc (default "doc")

fabric dockerAuth
-----------------

Docker login for pushing images

### Synopsis

Docker login for pushing images

    fabric dockerAuth <DockerRegistryURL> <User> <Password>

### Options

      -h, --help   help for dockerAuth
