package auth

import (
	"context"
	"crypto/subtle"
	"net/http"
)

type key int

const (
	// tenantsKey Key used to pass prometheus tenant id though the middleware context
	tenantsKey key = iota
	adminsKey
	realm = "Prometheus multi-tenant proxy"
)

// BasicAuth can be used as a middleware chain to authenticate users before proxying a request
func BasicAuth(handler http.Handler, authConfig *Authn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		authorized, tenants := isAuthorized(user, pass, authConfig)
		if !ok || !authorized {
			writeUnauthorisedResponse(w)
			return
		}
		ctx := context.WithValue(r.Context(), tenantsKey, tenants)
		ctx = context.WithValue(ctx, adminsKey, authConfig.Admins)
		handler.ServeHTTP(w, r.WithContext(ctx))
	}
}

func isAuthorized(user string, pass string, authConfig *Authn) (bool, []string) {
	for _, v := range authConfig.StaticUsers {
		if subtle.ConstantTimeCompare([]byte(user), []byte(v.Username)) == 1 && subtle.ConstantTimeCompare([]byte(pass), []byte(v.Password)) == 1 {
			return true, v.Tenants
		}
	}
	return false, nil
}

func writeUnauthorisedResponse(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`"`)
	w.WriteHeader(401)
	w.Write([]byte("Unauthorised\n"))
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
