package proxy

import (
	"bytes"
	"fmt"
	"io"
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
	ok, err := p.checkRequest(r)
	if err != nil {
		log.Printf("failed to check request: %v\n", err)
	}
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}
	p.reverseProxy.ServeHTTP(w, r)
	r.URL.User = nil
	log.Printf("[TO (%v)]\t%s %s %s\n", strings.Join(auth.TenantsFromCtx(r.Context()), ","), r.RemoteAddr, r.Method, r.URL)
}

func (p *prometheusProxy) modifyRequest(r *http.Request, prometheusQueryParameter string) error {
	// duplicate request body as non GET requests may read it to parse the form
	var original, discard bytes.Buffer
	if _, err := io.Copy(io.MultiWriter(&original, &discard), r.Body); err != nil {
		return err
	}
	// set the copy
	r.Body = io.NopCloser(&discard)
	tenants := auth.TenantsFromCtx(r.Context())
	matcher := &labels.Matcher{
		Name:  p.label,
		Type:  labels.MatchRegexp,
		Value: fmt.Sprintf("(%s)", strings.Join(tenants, "|")),
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
	// restore the original
	r.Body = io.NopCloser(&original)
	return nil
}

func (p *prometheusProxy) checkRequest(r *http.Request) (bool, error) {
	if auth.IsAdmin(r.Context()) {
		log.Println("skipping admin request")
		return true, nil
	}
	switch {
	case r.URL.Path == "/api/v1/query" || r.URL.Path == "/api/v1/query_range":
		if err := p.modifyRequest(r, "query"); err != nil {
			return true, err
		}
	case r.URL.Path == "/api/v1/series" || r.URL.Path == "/federate" || strings.HasPrefix(r.URL.Path, "/api/v1/label"):
		if err := p.modifyRequest(r, "match[]"); err != nil {
			return true, err
		}
	case strings.HasPrefix(r.URL.Path, "/api/v1"):
		return false, nil
	}

	r.Host = p.prometheusServerURL.Host
	r.URL.Scheme = p.prometheusServerURL.Scheme
	r.URL.Host = p.prometheusServerURL.Host
	r.Header.Set("X-Forwarded-Host", r.Host)
	p.ensureEndpointBasicAuth(r)
	return true, nil
}

func (p *prometheusProxy) ensureEndpointBasicAuth(r *http.Request) {
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
