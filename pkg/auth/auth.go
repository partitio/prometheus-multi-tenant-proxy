package auth

import (
	"context"
	"net/http"

	"github.com/k8spin/prometheus-multi-tenant-proxy/pkg/config"
)

type Method string

type key int

const (
	// tenantsKey Key used to pass prometheus tenant id though the middleware context
	tenantsKey key = iota
	adminsKey
	realm = "Prometheus multi-tenant proxy"
)

type ProviderFactory func(config *config.Authn) (Provider, error)

type Provider interface {
	Authenticate(handler http.Handler) http.HandlerFunc
}

type ProviderFunc func(handler http.Handler) http.HandlerFunc

func (f ProviderFunc) Authenticate(handler http.Handler) http.HandlerFunc {
	return f(handler)
}

func TenantsFromCtx(ctx context.Context) []string {
	return fromCtx(ctx, tenantsKey)
}

func AdminsFromCtx(ctx context.Context) []string {
	return fromCtx(ctx, adminsKey)
}

func IsAdmin(ctx context.Context) bool {
	for _, v := range AdminsFromCtx(ctx) {
		for _, vv := range TenantsFromCtx(ctx) {
			if v == vv {
				return true
			}
		}
	}
	return false
}

func fromCtx(ctx context.Context, key key) []string {
	v := ctx.Value(key)
	if ts, ok := v.([]string); ok {
		return ts
	}
	return nil
}
