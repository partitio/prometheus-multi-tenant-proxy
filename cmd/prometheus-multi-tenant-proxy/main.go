package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/k8spin/prometheus-multi-tenant-proxy/pkg/auth"
	proxy "github.com/k8spin/prometheus-multi-tenant-proxy/pkg/proxy"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	var (
		port int
		prom string
		config string
		label string
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
			Name:   "run",
			Usage:  "Runs the Prometheus multi-tenant proxy",
			Flags: []cli.Flag{
				&cli.IntFlag{
					Name:  "port",
					Usage: "Port to expose this prometheus proxy",
					Value: 9092,
					EnvVars: []string{"PROXY_PORT"},
					Destination: &port,
				}, &cli.StringFlag{
					Name:  "prometheus-endpoint",
					Usage: "Prometheus server endpoint",
					Value: "http://localhost:9091",
					EnvVars: []string{"PROMETHEUS_ENDPOINT"},
					Destination: &prom,
				}, &cli.StringFlag{
					Name:  "auth-config",
					Usage: "AuthN yaml configuration file path",
					Value: "authn.yaml",
					EnvVars: []string{"AUTH_CONFIG"},
					Destination: &config,
				}, &cli.StringFlag{
					Name: "tenant-label",
					Usage: "Label to match the tenant against",
					Value: "tenant",
					EnvVars: []string{"TENANT_LABEL"},
					Destination: &label,
				},
			},
			Action: func(_ *cli.Context) error {
				prometheusServerURL, err := url.Parse(prom)
				if err != nil {
					log.Fatalf("invalid prometheus endpoint: %v\n", err)
				}
				serveAt := fmt.Sprintf(":%d", port)
				authConfig, err := auth.ParseConfig(config)
				if err != nil {
					log.Fatalf("failed to parse config: %v\n", err)
				}
				handler := proxy.ReversePrometheus(prometheusServerURL, label)
				http.Handle("/-/healthy", LogRequest(handler))
				http.Handle("/-/ready", LogRequest(handler))
				http.Handle("/", LogRequest(auth.BasicAuth(handler, authConfig)))
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

