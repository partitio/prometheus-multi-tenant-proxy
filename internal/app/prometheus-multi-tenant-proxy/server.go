package proxy

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/angelbarrera92/prometheus-multi-tenant-proxy/internal/pkg"
	"github.com/urfave/cli"
)

// Serve serves
func Serve(c *cli.Context) error {
	prometheusLabelProxyServerURL, _ := url.Parse(c.String("prometheus-label-proxy-endpoint"))
	serveAt := fmt.Sprintf(":%d", c.Int("port"))
	authConfigLocation := c.String("auth-config")
	authConfig, _ := pkg.ParseConfig(&authConfigLocation)

	http.HandleFunc("/", createHandler(prometheusLabelProxyServerURL, authConfig))
	if err := http.ListenAndServe(serveAt, nil); err != nil {
		log.Fatalf("Prometheus multi tenant proxy can not start %v", err)
		return err
	}
	return nil
}

func createHandler(prometheusLabelProxyServerURL *url.URL, authConfig *pkg.Authn) http.HandlerFunc {
	reverseProxy := httputil.NewSingleHostReverseProxy(prometheusLabelProxyServerURL)
	return LogRequest(BasicAuth(ReversePrometheus(reverseProxy, prometheusLabelProxyServerURL), authConfig))
}