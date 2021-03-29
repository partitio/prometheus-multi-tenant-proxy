package auth

import (
	"context"
	"crypto/subtle"
	"net/http"
)

type key int

const (
	// Tenant Key used to pass prometheus tenant id though the middleware context
	Tenant key = iota
	realm      = "Prometheus multi-tenant proxy"
)

// BasicAuth can be used as a middleware chain to authenticate users before proxying a request
func BasicAuth(handler http.Handler, authConfig *Authn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		authorized, tenant := isAuthorized(user, pass, authConfig)
		if !ok || !authorized {
			writeUnauthorisedResponse(w)
			return
		}
		ctx := context.WithValue(r.Context(), Tenant, tenant)
		handler.ServeHTTP(w, r.WithContext(ctx))
	}
}

func isAuthorized(user string, pass string, authConfig *Authn) (bool, string) {
	for _, v := range authConfig.Users {
		if subtle.ConstantTimeCompare([]byte(user), []byte(v.Username)) == 1 && subtle.ConstantTimeCompare([]byte(pass), []byte(v.Password)) == 1 {
			return true, v.Tenant
		}
	}
	return false, ""
}

func writeUnauthorisedResponse(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`"`)
	w.WriteHeader(401)
	w.Write([]byte("Unauthorised\n"))
}
