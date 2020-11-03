package deploy

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const HTTP_POST = "POST"
const HTTP_GET = "GET"

type CortexClientV6 struct {
	Url     string
	Project string
	Token   string
}

type CortexClientV5 struct {
	Url     string
	Account string
	Token   string
}

type CortexAPI interface {
	GetURL() string
	GetToken() string
	GetAccount() string
	GetDockerRegistry() string
	DeployAction(filepath string) string
	DeployActionJson(actionType string, content []byte) string
	DeploySkill(filepath string) string
	DeploySkillJson(content []byte) string
	DeployAgent(filepath string) string
	DeployAgentJson(content []byte) string
	DeployDatasetJson(content []byte) string
	//DeploySnapshot(filepath string, actionImageMapping map[string]string) string
}

func NewCortexClient(url string, account string, user string, password string) CortexAPI {
	params := map[string]interface{}{"username": user, "password": password}
	body, _ := json.Marshal(params)
	client := &CortexClientV5{
		Url:     url,
		Account: account,
	}
	var result, error = post(client, fmt.Sprint("/v2/admin/", account, "/users/authenticate"), body)
	if error != nil {
		log.Fatalln(error)
	}
	client.Token = gjson.Get(string(result), "jwt").String()
	return client
}

func NewCortexClientExistingToken(url string, account string, token string) CortexAPI {
	client := &CortexClientV5{
		Url:     url,
		Account: account,
		Token:   token,
	}
	return client
}

func NewCortexClientPAT(project string, pat string) CortexAPI {
	bytes, err := ioutil.ReadFile(pat)
	if err != nil {
		log.Fatalln(err)
	}
	data := map[string]interface{}{}
	err = json.Unmarshal(bytes, &data)
	if err != nil {
		log.Fatalln(err)
	}

	client := &CortexClientV6{
		Url:     data["url"].(string),
		Project: project,
		Token:   generateJwt(data),
	}
	return client
}

func generateJwt(data map[string]interface{}) string {
	var set jose.JSONWebKey
	bytes, err := json.Marshal(data["jwk"])
	if err != nil {
		log.Fatalln(err)
	}
	if err := set.UnmarshalJSON([]byte(bytes)); err != nil {
		log.Fatalln(err)
	}

	key := jose.SigningKey{Algorithm: jose.EdDSA, Key: set}
	var signerOpts = jose.SignerOptions{}
	//signerOpts.WithBase64(true)
	signer, err := jose.NewSigner(key, &signerOpts)
	if err != nil {
		log.Fatalf("failed to create signer:%+v", err)
	}
	builder := jwt.Signed(signer)
	token, err := builder.Claims(&jwt.Claims{
		Issuer:  data["issuer"].(string),
		Subject: data["username"].(string),
		//ID:       "id1",
		Audience: jwt.Audience{data["audience"].(string)},
		IssuedAt: jwt.NewNumericDate(time.Now()),
		Expiry:   jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
	}).CompactSerialize()

	return token
}

//V5
func (c *CortexClientV5) GetURL() string {
	return c.Url
}

func (c *CortexClientV5) GetToken() string {
	return c.Token
}

func (c *CortexClientV5) GetAccount() string {
	return c.Account
}

func (c *CortexClientV5) GetDockerRegistry() string {
	var result, error = get(c, "/v3/actions/_config")
	if error != nil {
		log.Fatalln(error)
	}
	value := gjson.Get(string(result), "config.dockerPrivateRegistryUrl").String()
	return fmt.Sprint(value, "/", c.Account)
}

func (c *CortexClientV5) DeployAction(filepath string) string {
	content, err := ioutil.ReadFile(filepath)
	if err != nil {
		log.Fatalln(err)
	}
	actionType := gjson.Get(string(content), "actionType").String()
	return c.DeployActionJson(actionType, content)
}

func (c *CortexClientV5) DeployActionJson(actionType string, content []byte) string {
	var result, error = post(c, "/v3/actions?actionType="+actionType, content)
	if error != nil {
		log.Fatalln(error)
	}
	return string(result)
}

//https://github.com/CognitiveScale/cortex-cli/blob/6c91a3e94442f690c0de054545b9b214a17b6929/src/client/catalog.js#L42
func (c *CortexClientV5) DeploySkill(filepath string) string {
	content, err := ioutil.ReadFile(filepath)
	if err != nil {
		log.Fatalln(err)
	}
	return c.DeploySkillJson([]byte(content))
}

func (c *CortexClientV5) DeploySkillJson(content []byte) string {
	var result, error = post(c, "/v3/catalog/skills", content)
	if error != nil {
		log.Fatalln(error)
	}
	return string(result)
}

//https://github.com/CognitiveScale/cortex-cli/blob/6c91a3e94442f690c0de054545b9b214a17b6929/src/client/catalog.js#L139
func (c *CortexClientV5) DeployAgent(filepath string) string {
	content, err := ioutil.ReadFile(filepath)
	if err != nil {
		log.Fatalln(err)
	}
	return c.DeployAgentJson(content)
}

func (c *CortexClientV5) DeployAgentJson(content []byte) string {
	var result, error = post(c, "/v3/catalog/agents", content)
	if error != nil {
		log.Fatalln(error)
	}
	return string(result)
}

func (c *CortexClientV5) DeployDatasetJson(content []byte) string {
	var result, error = post(c, "/v3/datasets", content)
	if error != nil {
		log.Fatalln(error)
	}
	return string(result)
}

//V6
func (c *CortexClientV6) GetURL() string {
	return c.Url
}

func (c *CortexClientV6) GetToken() string {
	return c.Token
}

func (c *CortexClientV6) GetAccount() string {
	return c.Project
}

func (c *CortexClientV6) GetDockerRegistry() string {
	var result, error = get(c, "/v3/actions/_config")
	if error != nil {
		log.Fatalln(error)
	}
	value := gjson.Get(string(result), "config.dockerPrivateRegistryUrl").String()
	return fmt.Sprint(value, "/", c.Project)
}

func (c *CortexClientV6) DeployAction(filepath string) string {
	content, err := ioutil.ReadFile(filepath)
	if err != nil {
		log.Fatalln(err)
	}
	actionType := gjson.Get(string(content), "actionType").String()
	return c.DeployActionJson(actionType, content)
}

func (c *CortexClientV6) DeployActionJson(actionType string, content []byte) string {
	var result, error = post(c, "/fabric/v4/projects/"+c.Project+"/actions?actionType="+actionType, content)
	if error != nil {
		log.Fatalln(error)
	}
	return string(result)
}

//https://github.com/CognitiveScale/cortex-cli/blob/6c91a3e94442f690c0de054545b9b214a17b6929/src/client/catalog.js#L42
func (c *CortexClientV6) DeploySkill(filepath string) string {
	content, err := ioutil.ReadFile(filepath)
	if err != nil {
		log.Fatalln(err)
	}
	return c.DeploySkillJson([]byte(content))
}

func (c *CortexClientV6) DeploySkillJson(content []byte) string {
	var result, error = post(c, "/fabric/v4/projects/"+c.Project+"/skills", content)
	if error != nil {
		log.Fatalln(error)
	}
	return string(result)
}

//https://github.com/CognitiveScale/cortex-cli/blob/6c91a3e94442f690c0de054545b9b214a17b6929/src/client/catalog.js#L139
func (c *CortexClientV6) DeployAgent(filepath string) string {
	content, err := ioutil.ReadFile(filepath)
	if err != nil {
		log.Fatalln(err)
	}
	return c.DeployAgentJson(content)
}

func (c *CortexClientV6) DeployAgentJson(content []byte) string {
	var result, error = post(c, "/fabric/v4/projects/"+c.Project+"/agents", content)
	if error != nil {
		log.Fatalln(error)
	}
	return string(result)
}

func (c *CortexClientV6) DeployDatasetJson(content []byte) string {
	var result, error = post(c, "/fabric/v4/projects/"+c.Project+"/datasets", content)
	if error != nil {
		log.Fatalln(error)
	}
	return string(result)
}

// Common in v5 and v6
func DeploySnapshot(cortex CortexAPI, filepath string, actionImageMapping map[string]string) {
	content, err := ioutil.ReadFile(filepath)
	if err != nil {
		log.Fatalln("Failed to read Cortex Agent Snapshot file ", filepath, " Error: ", err)
	}
	if strings.HasSuffix(filepath, ".yaml") {
		content, err = yaml.YAMLToJSON(content)
		if err != nil {
			log.Fatalln("Failed to parse Cortex Agent Snapshot file ", filepath, " Error: ", err)
		}
	}
	snapshot := gjson.Parse(string(content))
	agent := snapshot.Get("agent")
	skills := snapshot.Get("dependencies.skills")
	actions := snapshot.Get("dependencies.actions")
	datasets := snapshot.Get("dependencies.datasets")

	datasets.ForEach(func(key, value gjson.Result) bool {
		logs := cortex.DeployDatasetJson([]byte(value.Raw))
		log.Println(logs)
		return true
	})

	skills.ForEach(func(key, value gjson.Result) bool {
		logs := cortex.DeploySkillJson([]byte(value.Raw))
		log.Println(logs)
		return true
	})

	actions.ForEach(func(key, value gjson.Result) bool {
		if actionImageMapping != nil {
			action := value.Map()
			imageName := DockerImageName(action["image"].String())
			image := actionImageMapping[imageName]
			if image != "" {
				//TODO - [2nd iteration] - evaluate better JSON substitution/templating. Need to support: variable substitution in connections,
				// support differ action config across env, ex.
				// 	higher resource limit (or cpu in dev vs gpu in prod) in prod compare to dev (podspec json substitution)
				//	higher scale count in prod (action config substitution)
				updated, _ := sjson.Set(value.Raw, "image", image)
				//parse podspec json into object before setting, for correct formatting
				podspec := value.Get("podSpec").String()
				var podspecDef []map[string]interface{}
				json.Unmarshal([]byte(podspec), &podspecDef)
				updated, _ = sjson.Set(updated, "podSpec", podspecDef)
				value = gjson.Parse(updated)
			} else {
				log.Println("[IMP] Docker image ", action["image"].String(), " used by action ", action["name"].String(), " is not built in this run, make sure it exists in docker registry")
			}
		}
		logs := cortex.DeployActionJson(value.Get("type").String(), []byte(value.Raw))
		log.Println(logs)
		return true
	})

	logs := cortex.DeployAgentJson([]byte(agent.Raw))
	log.Println(logs)
}

func get(cortex CortexAPI, path string) ([]byte, error) {
	return do(cortex, path, HTTP_GET, nil)
}

func post(cortex CortexAPI, path string, body []byte) ([]byte, error) {
	return do(cortex, path, HTTP_POST, body)
}

func do(cortex CortexAPI, path string, method string, body []byte) ([]byte, error) {
	url, err := url.Parse(cortex.GetURL() + path)
	if err != nil {
		log.Fatalln(err)
	}
	request := &http.Request{
		URL:    url,
		Method: method,
		Header: map[string][]string{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprint("Bearer ", cortex.GetToken())},
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
		error = errors.New(fmt.Sprint("URL ", url.String(), " failed with status ", response.StatusCode, " Error: ", string(data)))
	}
	defer response.Body.Close()
	return data, error
}

func DockerImageName(dockerTag string) string {
	splits := strings.Split(dockerTag, "/")
	return strings.Split(splits[len(splits)-1], ":")[0]
}
