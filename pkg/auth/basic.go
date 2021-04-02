package auth

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"

	"github.com/k8spin/prometheus-multi-tenant-proxy/pkg/config"
)

const (
	BasicMethod Method = "basic"
)

var (
	_ ProviderFactory = NewBasic
)

func NewBasic(config *config.Authn) (Provider, error) {
	if config == nil {
		return nil, errors.New("config cannot be nil")
	}
	return ProviderFunc(func(handler http.Handler) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			user, pass, ok := r.BasicAuth()
			authorized, tenants := isAuthorized(user, pass, config)
			if !ok || !authorized {
				writeBasicUnauthorisedResponse(w)
				return
			}
			ctx := context.WithValue(r.Context(), tenantsKey, tenants)
			ctx = context.WithValue(ctx, adminsKey, config.Admins)
			handler.ServeHTTP(w, r.WithContext(ctx))
		}
	}), nil
}

func isAuthorized(user string, pass string, authConfig *config.Authn) (bool, []string) {
	for _, v := range authConfig.StaticUsers {
		if subtle.ConstantTimeCompare([]byte(user), []byte(v.Username)) == 1 && subtle.ConstantTimeCompare([]byte(pass), []byte(v.Password)) == 1 {
			return true, v.Tenants
		}
	}
	return false, nil
}

func writeBasicUnauthorisedResponse(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`"`)
	w.WriteHeader(401)
	w.Write([]byte("Unauthorised\n"))
}
