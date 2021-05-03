package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/k8spin/prometheus-multi-tenant-proxy/pkg/auth"
	config2 "github.com/k8spin/prometheus-multi-tenant-proxy/pkg/config"
	proxy "github.com/k8spin/prometheus-multi-tenant-proxy/pkg/proxy"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	var (
		port            int
		prom            string
		alert           string
		config          string
		authMethod      string
		label           string
		enableLabelsAPI bool
	)
	app := cli.NewApp()
	app.Name = "Prometheus multi-tenant proxy"
	app.Usage = "Makes your Prometheus server multi tenant"
	app.Version = version
	app.Authors = []*cli.Author{
		{Name: "Angel Barrera", Email: "angel@k8spin.cloud"},
		{Name: "Pau Rosello", Email: "pau@k8spin.cloud"},
	}
	app.Commands = []*cli.Command{
		{
			Name:  "run",
			Usage: "Runs the Prometheus multi-tenant proxy",
			Flags: []cli.Flag{
				&cli.IntFlag{
					Name:        "port",
					Usage:       "Port to expose this prometheus proxy",
					Value:       9092,
					EnvVars:     []string{"PROXY_PORT"},
					Destination: &port,
				}, &cli.StringFlag{
					Name:        "prometheus-endpoint",
					Usage:       "Prometheus server endpoint",
					Value:       "http://localhost:9090",
					EnvVars:     []string{"PROMETHEUS_ENDPOINT"},
					Destination: &prom,
					Required:    true,
				},
				&cli.StringFlag{
					Name:        "alertmanager-endpoint",
					Usage:       "Alertmanager server endpoint, if defined requests will be proxied to alertmanager",
					EnvVars:     []string{"ALERTMANAGER_ENDPOINT"},
					Destination: &alert,
				},
				&cli.StringFlag{
					Name:        "auth-config",
					Usage:       "AuthN yaml configuration file path",
					Value:       "authn.yaml",
					EnvVars:     []string{"AUTH_CONFIG"},
					Destination: &config,
				}, &cli.StringFlag{
					Name:        "auth",
					Usage:       "Auth source: one of basic or oidc",
					Value:       "basic",
					EnvVars:     []string{"AUTH"},
					Destination: &authMethod,
				}, &cli.StringFlag{
					Name:        "tenant-label",
					Usage:       "Label to match the tenant against",
					Value:       "tenant",
					EnvVars:     []string{"TENANT_LABEL"},
					Destination: &label,
				}, &cli.BoolFlag{
					Name: "enable-label-apis",
					Usage: "When specified proxy allows to inject label to label APIs like /api/v1/labels and " +
						"/api/v1/label/<name>/values.\"+\n\t\t\"NOTE: Enable with care. Selection of matcher is still " +
						"in development, see https://github.com/thanos-io/thanos/issues/3351 and " +
						"https://github.com/prometheus/prometheus/issues/6178. If enabled and\"+\n\t\t\"any labels " +
						"endpoint does not support selectors, injected matcher will be silently dropped.",
					Value:       false,
					EnvVars:     []string{"ENABLE_LABEL_APIS"},
					Destination: &enableLabelsAPI,
				},
			},
			Action: func(_ *cli.Context) error {
				prometheusServerURL, err := url.Parse(prom)
				if err != nil {
					log.Fatalf("invalid prometheus endpoint: %v\n", err)
				}
				var alertmanagerServerURL *url.URL
				if alert != "" {
					if alertmanagerServerURL, err = url.Parse(alert); err != nil {
						log.Fatalf("invalid alertmanager endpoint: %v\n", err)
					}
				}
				serveAt := fmt.Sprintf(":%d", port)
				authConfig, err := config2.Parse(config)
				if err != nil {
					log.Fatalf("failed to parse config: %v\n", err)
				}
				var factory auth.ProviderFactory
				switch auth.Method(strings.ToLower(authMethod)) {
				case auth.BasicMethod:
					factory = auth.NewBasic
				case auth.OIDCMethod:
					factory = auth.NewOIDC
				default:
					log.Fatalf("unknown auth provider: %s", authMethod)
				}
				provider, err := factory(authConfig)
				if err != nil {
					log.Fatalf("failed to initalize auth provider %s: %v", authMethod, err)
				}
				handler, err := proxy.ReversePrometheus(
					proxy.WithPrometheus(prometheusServerURL),
					proxy.WithAlertmanager(alertmanagerServerURL),
					proxy.WithLabel(label),
					proxy.WithLabelsAPI(enableLabelsAPI),
				)
				if err != nil {
					log.Fatalf("init proxy: %v", err)
				}
				http.Handle("/-/healthy", LogRequest(handler))
				http.Handle("/-/ready", LogRequest(handler))
				http.Handle("/", LogRequest(provider.Authenticate(handler)))
				if err := http.ListenAndServe(serveAt, nil); err != nil {
					log.Fatalf("Prometheus multi tenant proxy can not start %v", err)
					return err
				}
				return nil
			},
		},
	}
	app.Run(os.Args)
}

// LogRequest can be used as a middleware chain to log every request before proxying the request
func LogRequest(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[FROM]\t%s %s %s\n", r.RemoteAddr, r.Method, r.URL)
		handler.ServeHTTP(w, r)
	})
}
