package proxy

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	injector "github.com/prometheus-community/prom-label-proxy/injectproxy"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/promql/parser"

	"github.com/k8spin/prometheus-multi-tenant-proxy/pkg/auth"
)

// ReversePrometheus a
func ReversePrometheus(prometheusServerURL *url.URL, label string) http.Handler {
	reverseProxy := httputil.NewSingleHostReverseProxy(prometheusServerURL)
	return &prometheusProxy{prometheusServerURL: prometheusServerURL, reverseProxy: reverseProxy, label: label}
}

type prometheusProxy struct {
	prometheusServerURL *url.URL
	reverseProxy        *httputil.ReverseProxy
	label               string
}

func (p *prometheusProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.checkRequest(r, p.prometheusServerURL)
	p.reverseProxy.ServeHTTP(w, r)
	log.Printf("[TO]\t%s %s %s\n", r.RemoteAddr, r.Method, r.URL)
}

func (p *prometheusProxy) modifyRequest(r *http.Request, prometheusQueryParameter string) error {
	tenant := r.Context().Value(auth.Tenant)
	matcher := &labels.Matcher{
		Name:  p.label,
		Type:  labels.MatchEqual,
		Value: tenant.(string),
	}
	var query string
	val := r.FormValue(prometheusQueryParameter)
	if val != "" {
		expr, err := parser.ParseExpr(val)
		if err != nil {
			return err
		}
		e := injector.NewEnforcer(matcher)
		if err := e.EnforceNode(expr); err != nil {
			return err
		}
		query = expr.String()
	} else {
		query = fmt.Sprintf("{%s}", matcher.String())
	}

	q := r.URL.Query()
	q.Set(prometheusQueryParameter, query)
	r.URL.RawQuery = q.Encode()
	return nil
}

func (p *prometheusProxy) checkRequest(r *http.Request, prometheusServerURL *url.URL) error {
	if r.URL.Path == "/api/v1/query" || r.URL.Path == "/api/v1/query_range" {
		if err := p.modifyRequest(r, "query"); err != nil {
			return err
		}
	}
	if r.URL.Path == "/api/v1/series" || r.URL.Path == "/federate" || strings.HasPrefix(r.URL.Path, "/api/v1/label") {
		if err := p.modifyRequest(r, "match[]"); err != nil {
			return err
		}
	}
	r.Host = prometheusServerURL.Host
	r.URL.Scheme = prometheusServerURL.Scheme
	r.URL.Host = prometheusServerURL.Host
	r.Header.Set("X-Forwarded-Host", r.Host)
	p.ensureBasicAuth(r)
	return nil
}

func (p *prometheusProxy) ensureBasicAuth(r *http.Request) {
	if p.prometheusServerURL.User == nil {
		return
	}
	pass, ok := p.prometheusServerURL.User.Password()
	if !ok {
		return
	}
	r.Header.Del("Authorization")
	r.SetBasicAuth(p.prometheusServerURL.User.Username(), pass)
	r.URL.User = p.prometheusServerURL.User
}
