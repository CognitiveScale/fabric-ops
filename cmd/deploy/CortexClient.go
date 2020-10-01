package deploy

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/tidwall/gjson"
	"io/ioutil"
	"net/http"
	"net/url"
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
	var result, _ = client.do(fmt.Sprint("/v2/admin/", account, "/users/authenticate"), HTTP_POST, body)
	jwt := gjson.Get(result, "jwt")
	client.Token = jwt.String()
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
	var result, _ = c.do("/v3/actions/_config", HTTP_GET, nil)
	value := gjson.Get(result, "config.dockerPrivateRegistryUrl")
	return value.String()
}

func (c *CortexClient) DeployAction(filepath string) string {
	content, _ := ioutil.ReadFile(filepath)
	actionType := gjson.Get(string(content), "actionType").String()
	var result, _ = c.do("/v3/actions?actionType="+actionType, HTTP_POST, content)
	return result
}

//https://github.com/CognitiveScale/cortex-cli/blob/6c91a3e94442f690c0de054545b9b214a17b6929/src/client/catalog.js#L42
func (c *CortexClient) DeploySkill(filepath string) string {
	content, _ := ioutil.ReadFile(filepath)
	var result, _ = c.do("/v3/catalog/skills", HTTP_POST, content)
	return result
}

//https://github.com/CognitiveScale/cortex-cli/blob/6c91a3e94442f690c0de054545b9b214a17b6929/src/client/catalog.js#L139
func (c *CortexClient) DeployAgent(filepath string) string {
	content, _ := ioutil.ReadFile(filepath)
	var result, _ = c.do("/v3/catalog/agents", HTTP_POST, content)
	return result
}

func (c *CortexClient) do(path string, method string, body []byte) (string, error) {
	url, _ := url.Parse(c.Url + path)

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
	var data, _ = ioutil.ReadAll(response.Body)
	if response.StatusCode > 201 {
		error = errors.New(string(data))
	}
	defer response.Body.Close()
	return string(data), error
}
