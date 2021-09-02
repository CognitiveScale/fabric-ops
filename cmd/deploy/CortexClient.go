package deploy

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const HTTP_POST = "POST"
const HTTP_DELETE = "DELETE"
const HTTP_GET = "GET"
const HTTP_PUT = "PUT"
const ARTIFACT_DIR = ".fabric"
const V6_BASE_URI = "/fabric/v4/projects/"

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
	DeployTypes(filepath string) string
	DeployTypesJson(content []byte) string
	DeployConnection(filepath string) string
	DeployConnectionJson(content []byte) string
}

func NewCortexClient(url string, account string, user string, password string) CortexAPI {
	params := map[string]interface{}{"username": user, "password": password}
	body, _ := json.Marshal(params)
	client := &CortexClientV5{
		Url:     url,
		Account: account,
	}
	var result, err = httpPost(client, fmt.Sprint("/v2/admin/", account, "/users/authenticate"), bytes.NewReader(body))
	if err != nil {
		log.Fatalln(err)
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
	content, err := ioutil.ReadFile(pat)
	if err != nil {
		log.Fatalln(err)
	}
	return NewCortexClientPATContent(project, content)
}

func NewCortexClientPATContent(project string, patToken []byte) CortexAPI {
	data := map[string]interface{}{}
	err := json.Unmarshal(patToken, &data)
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

//Generate JWT token from JWK for Cortex v6
func generateJwt(data map[string]interface{}) string {
	var set jose.JSONWebKey
	content, err := json.Marshal(data["jwk"])
	if err != nil {
		log.Fatalln(err)
	}
	if err := set.UnmarshalJSON(content); err != nil {
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
	var result, err = httpGet(c, "/v3/actions/_config")
	if err != nil {
		log.Fatalln(err)
	}
	value := gjson.Get(string(result), "config.dockerPrivateRegistryUrl").String()
	return fmt.Sprint(value, "/", c.Account)
}

func (c *CortexClientV5) DeployAction(filepath string) string {
	content, err := GetJsonContent(filepath)
	if err != nil {
		log.Fatalln(err)
	}
	actionType := gjson.Get(string(content), "actionType").String()
	return c.DeployActionJson(actionType, content)
}

func (c *CortexClientV5) DeployActionJson(actionType string, content []byte) string {
	var result, err = httpPost(c, "/v3/actions?actionType="+actionType, bytes.NewReader(content))
	if err != nil {
		log.Fatalln(err)
	}
	return string(result)
}

//https://github.com/CognitiveScale/cortex-cli/blob/6c91a3e94442f690c0de054545b9b214a17b6929/src/client/catalog.js#L42
func (c *CortexClientV5) DeploySkill(filepath string) string {
	content, err := GetJsonContent(filepath)
	if err != nil {
		log.Fatalln(err)
	}
	return c.DeploySkillJson(content)
}

func (c *CortexClientV5) DeploySkillJson(content []byte) string {
	var result, err = httpPost(c, "/v3/catalog/skills", bytes.NewReader(content))
	if err != nil {
		log.Fatalln(err)
	}
	return string(result)
}

//https://github.com/CognitiveScale/cortex-cli/blob/6c91a3e94442f690c0de054545b9b214a17b6929/src/client/catalog.js#L139
func (c *CortexClientV5) DeployAgent(filepath string) string {
	content, err := GetJsonContent(filepath)
	if err != nil {
		log.Fatalln(err)
	}
	return c.DeployAgentJson(content)
}

func (c *CortexClientV5) DeployAgentJson(content []byte) string {
	var result, err = httpPost(c, "/v3/catalog/agents", bytes.NewReader(content))
	if err != nil {
		log.Fatalln(err)
	}
	return string(result)
}

func (c *CortexClientV5) DeployDatasetJson(content []byte) string {
	var result, err = httpPost(c, "/v3/datasets", bytes.NewReader(content))
	if err != nil {
		log.Fatalln(err)
	}
	return string(result)
}

func (c *CortexClientV5) DeployTypes(filepath string) string {
	content, err := GetJsonContent(filepath)
	if err != nil {
		log.Fatalln(err)
	}
	return c.DeployTypesJson(content)
}

func (c *CortexClientV5) DeployTypesJson(content []byte) string {
	var result, err = httpPost(c, "/v3/catalog/types", bytes.NewReader(content))
	if err != nil {
		log.Fatalln(err)
	}
	return string(result)
}

func (c *CortexClientV5) DeployConnection(filepath string) string {
	content, err := GetJsonContent(filepath)
	if err != nil {
		log.Fatalln(err)
	}
	return c.DeployConnectionJson(content)
}

func (c *CortexClientV5) DeployConnectionJson(content []byte) string {
	var result, err = httpPost(c, "/v2/connections", bytes.NewReader(content))
	if err != nil {
		log.Fatalln(err)
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

// TODO update V6 Docker registry logic as per updated Action deployment (when ready)
func (c *CortexClientV6) GetDockerRegistry() string {
	var result, err = httpGet(c, "/v3/actions/_config")
	if err != nil {
		log.Fatalln(err)
	}
	value := gjson.Get(string(result), "config.dockerPrivateRegistryUrl").String()
	return fmt.Sprint(value, "/", c.Project)
}

func (c *CortexClientV6) DeployAction(filepath string) string {
	content, err := GetJsonContent(filepath)
	if err != nil {
		log.Fatalln(err)
	}
	actionType := gjson.Get(string(content), "actionType").String()
	return c.DeployActionJson(actionType, content)
}

func (c *CortexClientV6) DeployActionJson(actionType string, content []byte) string {
	var result, err = httpPost(c, V6_BASE_URI+c.Project+"/actions?actionType="+actionType, bytes.NewReader(content))
	if err != nil {
		log.Fatalln(err)
	}
	return string(result)
}

//https://github.com/CognitiveScale/cortex-cli/blob/6c91a3e94442f690c0de054545b9b214a17b6929/src/client/catalog.js#L42
func (c *CortexClientV6) DeploySkill(filepath string) string {
	content, err := GetJsonContent(filepath)
	if err != nil {
		log.Fatalln(err)
	}
	return c.DeploySkillJson(content)
}

func (c *CortexClientV6) DeploySkillJson(content []byte) string {
	var result, err = httpPost(c, V6_BASE_URI+c.Project+"/skills", bytes.NewReader(content))
	if err != nil {
		log.Fatalln(err)
	}
	return string(result)
}

//https://github.com/CognitiveScale/cortex-cli/blob/6c91a3e94442f690c0de054545b9b214a17b6929/src/client/catalog.js#L139
func (c *CortexClientV6) DeployAgent(filepath string) string {
	content, err := GetJsonContent(filepath)
	if err != nil {
		log.Fatalln(err)
	}
	return c.DeployAgentJson(content)
}

func (c *CortexClientV6) DeployAgentJson(content []byte) string {
	var result, err = httpPost(c, V6_BASE_URI+c.Project+"/agents", bytes.NewReader(content))
	if err != nil {
		log.Fatalln(err)
	}
	return string(result)
}

func (c *CortexClientV6) DeployDatasetJson(content []byte) string {
	var result, err = httpPost(c, V6_BASE_URI+c.Project+"/datasets", bytes.NewReader(content))
	if err != nil {
		log.Fatalln(err)
	}
	return string(result)
}

func (c *CortexClientV6) DeployTypes(filepath string) string {
	content, err := GetJsonContent(filepath)
	if err != nil {
		log.Fatalln(err)
	}
	return c.DeployTypesJson(content)
}

func (c *CortexClientV6) DeployTypesJson(content []byte) string {
	var result, err = httpPost(c, V6_BASE_URI+c.Project+"/types", bytes.NewReader(content))
	if err != nil {
		log.Fatalln(err)
	}
	return string(result)
}

func (c *CortexClientV6) DeployConnection(filepath string) string {
	content, err := GetJsonContent(filepath)
	if err != nil {
		log.Fatalln(err)
	}
	return c.DeployConnectionJson(content)
}

func (c *CortexClientV6) DeployConnectionJson(content []byte) string {
	var result, err = httpPost(c, V6_BASE_URI+c.Project+"/connections", bytes.NewReader(content))
	if err != nil {
		log.Fatalln(err)
	}
	return string(result)
}

func GetJsonContent(filepath string) ([]byte, error) {
	content, err := ioutil.ReadFile(filepath)
	if err != nil {
		return content, err
	}
	if strings.HasSuffix(filepath, ".yaml") || strings.HasSuffix(filepath, ".yml") {
		content, err = yaml.YAMLToJSON(content)
	}
	return content, err
}

func DeployCampaign(cortex CortexClientV6, filename string, deployable bool, overwrite bool) error {
	campaignUrl := V6_BASE_URI + cortex.Project + "/campaigns/import?deployable=" + strconv.FormatBool(deployable) + "&overwrite=" + strconv.FormatBool(overwrite)
	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)

	fileWriter, err := bodyWriter.CreateFormFile("file", filename)
	if err != nil {
		log.Println("error creating form data for file upload")
		return err
	}

	fh, err := os.Open(filename)
	if err != nil {
		log.Println("error opening campaign zip file")
		return err
	}
	defer fh.Close()

	_, err = io.Copy(fileWriter, fh)
	if err != nil {
		log.Println(err)
		return err
	}

	contentType := bodyWriter.FormDataContentType()
	bodyWriter.Close()

	resp, err := fileUpload(&cortex, campaignUrl, bodyBuf, contentType, HTTP_POST)
	if err != nil {
		log.Println(string(resp))
		return err
	}
	var prettyJSON bytes.Buffer
	err = json.Indent(&prettyJSON, resp, "", "    ")
	if err == nil {
		log.Println("Campaign "+strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))+" deployment status: ", string(prettyJSON.Bytes()))
	}
	return err
}

func DeployModel(cortex CortexClientV6, filepath string) string {
	content, err := GetJsonContent(filepath)
	if err != nil {
		log.Fatalln(err)
	}
	model := gjson.Parse(string(content))
	status := model.Get("status").String()
	// models can't be created with Published status, but exported only if published. So creating without status (using default initial status) and saving again with Published status
	if status == "Published" {
		modelBody := model.Value().(map[string]interface{})
		modelBody["status"] = "In development"
		initial, _ := json.Marshal(modelBody)
		res, err := httpPost(&cortex, V6_BASE_URI+cortex.Project+"/models", bytes.NewReader(initial))
		if err != nil {
			log.Println(string(res))
			log.Fatalln(err)
		}
	}
	res, err := httpPost(&cortex, V6_BASE_URI+cortex.Project+"/models", bytes.NewReader(content))
	if err != nil {
		log.Println(string(res))
		log.Fatalln(err)
	}
	return string(res)
}

func DeployExperiment(cortex CortexClientV6, filepath string) string {
	content, err := GetJsonContent(filepath)
	if err != nil {
		log.Fatalln(err)
	}
	res, err := httpPost(&cortex, V6_BASE_URI+cortex.Project+"/experiments", bytes.NewReader(content))
	if err != nil {
		log.Println(string(res))
		log.Fatalln(err)
	}
	return string(res)
}

func DeployExperimentRun(cortex CortexClientV6, filename string, repoDir string) string {
	content, err := GetJsonContent(filename)
	if err != nil {
		log.Fatalln(err)
	}
	run := gjson.Parse(string(content))
	expName := run.Get("experimentName").String()
	runId := run.Get("runId").String()
	artifacts := run.Get("artifacts")
	path := V6_BASE_URI + cortex.Project + "/experiments/" + url.PathEscape(expName) + "/runs"
	// experiment run is not upsert API, so deleting and inserting
	httpDelete(&cortex, path+"/"+runId)
	res, err := httpPost(&cortex, path, bytes.NewReader(content))
	if err != nil {
		log.Println(string(res))
		log.Fatalln(err)
	}
	if artifacts.Exists() {
		for k, v := range artifacts.Value().(map[string]interface{}) {
			artifactFile := filepath.Join(repoDir, ARTIFACT_DIR, v.(string))
			body, err := os.Open(artifactFile)
			if err != nil {
				log.Fatal("Failed to read Model artifact file ", artifactFile)
			}
			msg, err := fileUpload(&cortex, path+"/"+runId+"/artifacts/"+k, body, "application/octet-stream", HTTP_PUT)
			if err != nil {
				log.Fatalln(string(msg), err)
			}
		}
	}
	return string(res)
}

// Common in v5 and v6
func DeploySnapshot(cortex CortexAPI, filepath string, actionImageMapping map[string]string) {
	content, err := GetJsonContent(filepath)
	if err != nil {
		log.Fatalln("Failed to read Cortex Agent Snapshot file ", filepath, " Error: ", err)
	}
	snapshot := gjson.Parse(string(content))
	agent := snapshot.Get("agent")
	skills := snapshot.Get("dependencies.skills")
	actions := snapshot.Get("dependencies.actions")
	datasets := snapshot.Get("dependencies.datasets")
	types := snapshot.Get("dependencies.types")

	types.ForEach(func(key, value gjson.Result) bool {
		logs := cortex.DeployTypesJson([]byte(value.Raw))
		log.Println(logs)
		return true
	})

	datasets.ForEach(func(key, value gjson.Result) bool {
		logs := cortex.DeployDatasetJson([]byte(value.Raw))
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

	skills.ForEach(func(key, value gjson.Result) bool {
		logs := cortex.DeploySkillJson([]byte(value.Raw))
		log.Println(logs)
		return true
	})

	logs := cortex.DeployAgentJson([]byte(agent.Raw))
	log.Println(logs)
}

func httpGet(cortex CortexAPI, path string) ([]byte, error) {
	return do(cortex, path, HTTP_GET, nil, "application/json")
}

func httpPost(cortex CortexAPI, path string, body io.Reader) ([]byte, error) {
	return do(cortex, path, HTTP_POST, body, "application/json")
}

func httpDelete(cortex CortexAPI, path string) ([]byte, error) {
	return do(cortex, path, HTTP_DELETE, nil, "application/json")
}

func fileUpload(cortex CortexAPI, path string, body io.Reader, contentType string, method string) ([]byte, error) {
	return do(cortex, path, method, body, contentType)
}

var client *http.Client

func setupHttpClient() *http.Client {
	config := &tls.Config{}

	var ignoreCert, _ = strconv.ParseBool(GetEnvVar("IGNORE_INVALID_SSL_CERT"))
	if ignoreCert {
		config.InsecureSkipVerify = true
	}
	var sslCertsPath = GetEnvVar("SSL_CERTS_DIR")
	if sslCertsPath != "" {
		rootCAs, _ := x509.SystemCertPool()
		if rootCAs == nil {
			rootCAs = x509.NewCertPool()
			log.Println("SystemCertPool is not initialized. Using NewCertPool")
		}

		files, err := ioutil.ReadDir(sslCertsPath)
		if err != nil {
			log.Fatalf("Failed to read certs from %s : %v", sslCertsPath, err)
		}
		for _, file := range files {
			if file.IsDir() {
				continue
			}
			certFilePath := filepath.Join(sslCertsPath, file.Name())
			certs, err := ioutil.ReadFile(certFilePath)
			if err != nil {
				log.Fatalln("Failed to add cert ", certFilePath, err)
			}
			log.Printf("Adding SSL Cert %s to trusted root", certFilePath)
			if ok := rootCAs.AppendCertsFromPEM(certs); !ok {
				log.Printf("Failed to add SSL Cert %s to trusted root", certFilePath)
			}
		}
		config.RootCAs = rootCAs
	}
	var client = &http.Client{Transport: &http.Transport{TLSClientConfig: config}}
	return client

}

func do(cortex CortexAPI, path string, method string, body io.Reader, contentType string) ([]byte, error) {
	serviceUrl, err := url.Parse(cortex.GetURL() + path)
	if err != nil {
		log.Fatalln(err)
	}
	request := &http.Request{
		URL:    serviceUrl,
		Method: method,
		Header: map[string][]string{
			"Content-Type":  {contentType},
			"Authorization": {fmt.Sprint("Bearer ", cortex.GetToken())},
		},
	}
	if body != nil {
		request.Body = ioutil.NopCloser(body)
	}
	//lazy initialize
	if client == nil {
		client = setupHttpClient()
	}
	response, e := client.Do(request)
	if e != nil {
		//errors like connection refused, address not found etc
		return nil, e
	}
	var data, _ = ioutil.ReadAll(response.Body)
	if response.StatusCode > 201 {
		e = errors.New(fmt.Sprint("URL ", serviceUrl.String(), " failed with status ", response.StatusCode, " Error: ", string(data)))
	}
	defer response.Body.Close()
	return data, e
}

func DockerImageName(dockerTag string) string {
	splits := strings.Split(dockerTag, "/")
	return strings.Split(splits[len(splits)-1], ":")[0]
}

//replaced viper.GetString to remove vulnerable dependencies FAB-789 and FAB-792

func GetEnvVar(varname string) string {
	return os.Getenv(varname)
}
