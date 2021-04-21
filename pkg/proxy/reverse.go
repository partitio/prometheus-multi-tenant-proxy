package proxy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/prometheus-community/prom-label-proxy/injectproxy"

	"github.com/k8spin/prometheus-multi-tenant-proxy/pkg/auth"
)

var (
	passthrough = []string {
		"/api/v2/receivers",
		"/api/v2/groups",
	}
)

// ReversePrometheus
func ReversePrometheus(prometheusServerURL *url.URL, alertmanagerServerURL *url.URL, label string) (http.Handler, error) {
	if prometheusServerURL == nil {
		return nil, errors.New("prometheus server url must be not nil")
	}
	var proxyURL *url.URL
	switch {
	case alertmanagerServerURL != nil:
		proxyURL = alertmanagerServerURL
	default:
		proxyURL = prometheusServerURL
	}
	reverseProxy := httputil.NewSingleHostReverseProxy(proxyURL)
	reverseProxy.ModifyResponse = func(w *http.Response) error {
		// remove all cors headers to prevent duplicated header when proxied
		for _, v := range []string{"Access-Control-Allow-Origin", "Access-Control-Allow-Methods", "Access-Control-Allow-Headers"} {
			w.Header.Del(v)
		}
		return nil
	}
	labelProxy, err := injectproxy.NewRoutes(proxyURL, label, injectproxy.WithEnabledLabelsAPI(), injectproxy.WithPassthroughPaths(passthrough))
	if err != nil {
		return nil, err
	}
	cache := cache.New(5*time.Second, 10*time.Second)
	proxy := &prometheusProxy{
		prometheusServerURL: prometheusServerURL,
		reverseProxy:        reverseProxy,
		labelProxy:          labelProxy,
		label:               label,
		cache:               cache,
	}
	return proxy, nil
}

type prometheusProxy struct {
	prometheusServerURL   *url.URL
	alertmanagerServerURL *url.URL
	reverseProxy          *httputil.ReverseProxy
	labelProxy            http.Handler
	label                 string
	cache                 *cache.Cache
	mu                    sync.RWMutex
}

func (p *prometheusProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if auth.IsAdmin(r.Context()) {
		log.Println("skipping admin request")
		p.reverseProxy.ServeHTTP(w, r)
		return
	}
	if !strings.HasPrefix(r.URL.Path, "/api") && !strings.HasPrefix(r.URL.Path, "/federate") {
		log.Println("skipping static content request")
		p.reverseProxy.ServeHTTP(w, r)
		return
	}
	tenant, err := p.tenant(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !isPassthrough(r.URL) {
		q := r.URL.Query()
		q.Set(p.label, tenant)
		r.URL.RawQuery = q.Encode()
	}
	p.ensureEndpointBasicAuth(r)
	p.labelProxy.ServeHTTP(w, r)
	r.URL.User = nil
	log.Printf("[TO (%v)]\t%s %s %s\n", tenant, r.RemoteAddr, r.Method, r.URL)
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

func (p *prometheusProxy) tenant(ctx context.Context) (string, error) {
	tenantValues, err := p.labelValues()
	if err != nil {
		return "", err
	}
	tenants := auth.TenantsFromCtx(ctx)
	for _, v := range tenantValues {
		for _, vv := range tenants {
			if strings.ToLower(vv) == strings.ToLower(v) {
				return v, nil
			}
		}
	}
	return "partitio", nil
}

const (
	values = "values"
)

func (p *prometheusProxy) labelValues() ([]string, error) {
	p.mu.RLock()
	if v, ok := p.cache.Get(values); ok {
		p.mu.RUnlock()
		return v.([]string), nil
	}
	p.mu.RUnlock()
	p.mu.Lock()
	defer p.mu.Unlock()
	res, err := http.Get(fmt.Sprintf("%s/api/v1/label/%s/values", p.prometheusServerURL, p.label))
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected response status: %v", res.Status)
	}
	type promRes struct {
		Status string   `json:"status"`
		Data   []string `json:"data"`
	}
	r := &promRes{}
	if err := json.NewDecoder(res.Body).Decode(r); err != nil {
		return nil, err
	}
	p.cache.Set(values, r.Data, 5*time.Second)
	return r.Data, nil
}

func isPassthrough(url *url.URL) bool {
	if url == nil {
		return false
	}
	for _, v := range passthrough {
		if strings.HasPrefix(url.Path, v) {
			return true
		}
	}
	return false
}
