package deploy

import (
	"bytes"
	"github.com/fatih/color"
	"github.com/google/go-jsonnet"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"
)

var ScriptGenerator = template.Must(template.New("jsonnet").Parse(`
	local resource = import '{{.resourcePath}}';
	
	{{.content}}
	`))

// Transform
//Apply transformation on exported cortex resources in json/yaml as:
//	Add the json files to be transformed as import in the jsonnet provided in variable names `object`. Convert yaml to json
//	Add all env var, project, kind, artifactsDir and manifestFile as ext var and can be read in script by `std.extVar`
//	Return output JSON to be imported to target env
func Transform(resourceFile string, scriptPath string, kind string, artifactsDir string, manifestFile string) (string, error) {
	vm := jsonnet.MakeVM()
	vm.ErrorFormatter.SetColorFormatter(color.New(color.FgRed).Fprintf)

	for _, element := range os.Environ() {
		variable := strings.Split(element, "=")
		vm.ExtVar(variable[0], variable[1])
	}
	vm.ExtVar("kind", kind)
	vm.ExtVar("artifactsDir", artifactsDir)
	vm.ExtVar("manifestFile", manifestFile)

	resourcePath := GetResourceAsJson(resourceFile, artifactsDir)
	defer os.Remove(resourcePath)
	content, err := GetJsonContent(scriptPath)
	if err != nil {
		log.Fatal(err)
	}
	data := map[string]interface{}{
		"resourcePath": resourcePath,
		"content":      string(content),
	}
	buf := &bytes.Buffer{}
	if err := ScriptGenerator.Execute(buf, data); err != nil {
		log.Fatal(err)
	}
	script := buf.String()
	return vm.EvaluateAnonymousSnippet(scriptPath, script)
}

func GetResourceAsJson(resourceFile string, artifactsDir string) string {
	resource, err := GetJsonContent(resourceFile)
	if err != nil {
		log.Fatal(err)
	}
	resourcePath := filepath.Join(artifactsDir, "_tmp", strings.Replace(resourceFile, artifactsDir, "", 1)) + ".json"
	WriteToPath(resourcePath, resource)
	return resourcePath
}

func WriteToPath(resourcePath string, content []byte) {
	err := os.MkdirAll(path.Dir(resourcePath), 0755)
	if err != nil {
		log.Fatalln("Failed to write transformed resource", resourcePath, err)
	}
	err = ioutil.WriteFile(resourcePath, content, 0755)
	if err != nil {
		log.Fatalln("Failed to write transformed resource", resourcePath, err)
	}
}
