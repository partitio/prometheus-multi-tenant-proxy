module github.com/k8spin/prometheus-multi-tenant-proxy

go 1.15

require (
	github.com/coreos/go-oidc/v3 v3.0.0
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/prometheus-community/prom-label-proxy v0.3.0
	github.com/urfave/cli/v2 v2.3.0
	gopkg.in/yaml.v2 v2.4.0
)

replace github.com/prometheus-community/prom-label-proxy => github.com/partitio/prom-label-proxy v0.2.1-0.20210420131032-0e7456c057d6
