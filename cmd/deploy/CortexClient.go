package deploy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

const HTTP_POST = "POST"
const HTTP_GET = "GET"

type CortexClient struct {
	url     string
	account string
	token   string
}

func NewCortexClient(url string, account string, user string, password string) CortexClient {
	params := map[string]interface{}{"username": user, "password": password}
	client := CortexClient{
		url:     url,
		account: account,
	}
	var result = client.do(fmt.Sprint("/v2/admin/", account, "/users/authenticate"), HTTP_POST, params)
	client.token = fmt.Sprint(result["jwt"])
	return client
}

func (c *CortexClient) do(path string, method string, params map[string]interface{}) map[string]interface{} {
	url, _ := url.Parse(c.url + path)
	body, _ := json.Marshal(params)

	request := &http.Request{
		URL:    url,
		Method: method,
		Header: map[string][]string{
			"Content-Type":  {"application/json"},
			"authorization": {fmt.Sprint("Bearer ", c.token)},
		},
		Body: ioutil.NopCloser(bytes.NewReader(body)),
	}
	response, error := http.DefaultClient.Do(request)
	if error != nil {

	}
	var result map[string]interface{}
	json.NewDecoder(response.Body).Decode(&result)
	defer response.Body.Close()
	return result
}
