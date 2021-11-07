module fabric-ops

go 1.15

require (
	github.com/fatih/color v1.9.0
	github.com/ghodss/yaml v1.0.0
	github.com/go-git/go-git/v5 v5.2.0
	github.com/google/go-jsonnet v0.17.0
	github.com/spf13/cobra v1.1.1
	github.com/tidwall/gjson v1.11.0
	github.com/tidwall/sjson v1.1.7
	gopkg.in/square/go-jose.v2 v2.5.1
	gopkg.in/yaml.v2 v2.2.8
)

replace github.com/coreos/etcd => github.com/coreos/etcd v3.3.25+incompatible

replace github.com/hashicorp/consul/api => github.com/hashicorp/consul/api v1.8.1
