package deploy

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"strings"
)

const HTTP_POST = "POST"
const HTTP_GET = "GET"

type CortexClient struct {
	Url     string
	Account string
	Token   string
}

func NewCortexClient(url string, account string, user string, password string) CortexClient {
	params := map[string]interface{}{"username": user, "password": password}
	body, _ := json.Marshal(params)
	client := CortexClient{
		Url:     url,
		Account: account,
	}
	var result, error = client.post(fmt.Sprint("/v2/admin/", account, "/users/authenticate"), body)
	if error != nil {
		log.Fatalln(error)
	}
	client.Token = gjson.Get(string(result), "jwt").String()
	return client
}

func NewCortexClientExistingToken(url string, account string, token string) CortexClient {
	client := CortexClient{
		Url:     url,
		Account: account,
		Token:   token,
	}
	return client
}

func (c *CortexClient) GetDockerRegistry() string {
	var result, error = c.get("/v3/actions/_config")
	if error != nil {
		log.Fatalln(error)
	}
	value := gjson.Get(string(result), "config.dockerPrivateRegistryUrl").String()
	return fmt.Sprint(value, "/", c.Account)
}

func (c *CortexClient) DeployAction(filepath string) string {
	content, err := ioutil.ReadFile(filepath)
	if err != nil {
		log.Fatalln(err)
	}
	actionType := gjson.Get(string(content), "actionType").String()
	return c.DeployActionJson(actionType, content)
}

func (c *CortexClient) DeployActionJson(actionType string, content []byte) string {
	var result, error = c.post("/v3/actions?actionType="+actionType, content)
	if error != nil {
		log.Fatalln(error)
	}
	return string(result)
}

//https://github.com/CognitiveScale/cortex-cli/blob/6c91a3e94442f690c0de054545b9b214a17b6929/src/client/catalog.js#L42
func (c *CortexClient) DeploySkill(filepath string) string {
	content, err := ioutil.ReadFile(filepath)
	if err != nil {
		log.Fatalln(err)
	}
	return c.DeploySkillJson([]byte(content))
}

func (c *CortexClient) DeploySkillJson(content []byte) string {
	var result, error = c.post("/v3/catalog/skills", content)
	if error != nil {
		log.Fatalln(error)
	}
	return string(result)
}

//https://github.com/CognitiveScale/cortex-cli/blob/6c91a3e94442f690c0de054545b9b214a17b6929/src/client/catalog.js#L139
func (c *CortexClient) DeployAgent(filepath string) string {
	content, err := ioutil.ReadFile(filepath)
	if err != nil {
		log.Fatalln(err)
	}
	return c.DeployAgentJson(content)
}

func (c *CortexClient) DeployAgentJson(content []byte) string {
	var result, error = c.post("/v3/catalog/agents", content)
	if error != nil {
		log.Fatalln(error)
	}
	return string(result)
}

func (c *CortexClient) DeployDatasetJson(content []byte) string {
	var result, error = c.post("/v3/datasets", content)
	if error != nil {
		log.Fatalln(error)
	}
	return string(result)
}

func (c *CortexClient) DeploySnapshot(filepath string, actionImageMapping map[string]string) {
	content, _ := ioutil.ReadFile(filepath)
	jsonBytes, _ := yaml.YAMLToJSON(content)
	snapshot := gjson.Parse(string(jsonBytes))
	agent := snapshot.Get("agent")
	skills := snapshot.Get("dependencies.skills")
	actions := snapshot.Get("dependencies.actions")
	datasets := snapshot.Get("dependencies.datasets")

	datasets.ForEach(func(key, value gjson.Result) bool {
		logs := c.DeployDatasetJson([]byte(value.Raw))
		log.Println(logs)
		return true
	})

	skills.ForEach(func(key, value gjson.Result) bool {
		logs := c.DeploySkillJson([]byte(value.Raw))
		log.Println(logs)
		return true
	})

	actions.ForEach(func(key, value gjson.Result) bool {
		if actionImageMapping != nil {
			action := value.Map()
			image := DockerImageName(action["image"].String())
			image = actionImageMapping[image]
			if image != "" {
				//action["image"] = image
				updated, _ := sjson.Set(value.Raw, "image", image)
				value = gjson.Parse(updated)
				fmt.Println(value.Raw)
				fmt.Println(image)
			}
		}
		logs := c.DeployActionJson(value.Get("type").String(), []byte(value.Raw))
		log.Println(logs)
		return true
	})

	logs := c.DeployAgentJson([]byte(agent.Raw))
	log.Println(logs)
}

func (c *CortexClient) ExportAgents(exportDir string, agentNames ...string) {
	for _, agentName := range agentNames {
		res, _ := c.get("/v3/catalog/agents/" + agentName)
		agentJson := gjson.Parse(string(res))
		skills := agentJson.Get("skills.#.skillName").Array()
		var actions []string

		for _, skillNode := range skills {
			skillName := skillNode.String()

			res, _ = c.get("/v3/catalog/skills/" + skillName)
			//START remove internal fields
			var skillDef map[string]interface{}
			json.Unmarshal(res, &skillDef)
			clean(skillDef)
			skillDefClean, _ := json.Marshal(skillDef)
			println(string(skillDefClean))
			//END remove internal fields
			skillJson := gjson.Parse(string(res))
			skillJson.Get("inputs.#.routing.*.action").ForEach(func(key, value gjson.Result) bool {
				actions = append(actions, value.String())
				return true
			})
			//c.post("/v3/catalog/skills/", []byte(skillJson.String()))
		}
	}
}

func (c *CortexClient) getWithBody(path string, body []byte) ([]byte, error) {
	return c.do(path, HTTP_GET, body)
}

func (c *CortexClient) get(path string) ([]byte, error) {
	return c.do(path, HTTP_GET, nil)
}

func (c *CortexClient) post(path string, body []byte) ([]byte, error) {
	return c.do(path, HTTP_POST, body)
}

func (c *CortexClient) do(path string, method string, body []byte) ([]byte, error) {
	url, err := url.Parse(c.Url + path)
	if err != nil {
		log.Fatalln(err)
	}
	request := &http.Request{
		URL:    url,
		Method: method,
		Header: map[string][]string{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprint("Bearer ", c.Token)},
		},
	}
	if body != nil {
		request.Body = ioutil.NopCloser(bytes.NewReader(body))
	}
	response, error := http.DefaultClient.Do(request)
	if error != nil {
		//errors like connection refused, address not found etc
		return nil, error
	}
	var data, _ = ioutil.ReadAll(response.Body)
	if response.StatusCode > 201 {
		error = errors.New(string(data))
	}
	defer response.Body.Close()
	return data, nil
}

func clean(obj map[string]interface{}) {
	for key, val := range obj {
		if skipKey(key) {
			delete(obj, key)
			continue
		}
		switch val.(type) {
		case []interface{}:
			for _, item := range val.([]interface{}) {
				if reflect.ValueOf(item).Kind() == reflect.Map {
					clean(item.(map[string]interface{}))
				}
			}
		case map[string]interface{}:
			clean(val.(map[string]interface{}))
		}
	}
}

func skipKey(key string) bool {
	return strings.HasPrefix(key, "_")
}

func DockerImageName(dockerTag string) string {
	splits := strings.Split(dockerTag, "/")
	return strings.Split(splits[len(splits)-1], ":")[0]
}
