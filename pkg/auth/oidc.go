package auth

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"

	"github.com/k8spin/prometheus-multi-tenant-proxy/pkg/config"
)

const (
	OIDCMethod Method = "oidc"
)

var (
	_ ProviderFactory = NewOIDC
)

func NewOIDC(config *config.Authn) (Provider, error) {
	if config == nil {
		return nil, errors.New("config cannot be nil")
	}
	if config.OIDC == nil {
		return nil, errors.New("no oidc config provided")
	}
	// Initialize a provider by specifying dex's issuer URL.
	provider, err := oidc.NewProvider(context.Background(), config.OIDC.IssuerURL)
	if err != nil {
		return nil, err
	}
	// Create an ID token parser, but only trust ID tokens issued to "example-app"
	idTokenVerifier := provider.Verifier(&oidc.Config{ClientID: config.OIDC.ClientID})
	return &oidcProvider{config: config, verifier: idTokenVerifier}, nil
}

type oidcProvider struct {
	config   *config.Authn
	verifier *oidc.IDTokenVerifier
}

func (o *oidcProvider) Authenticate(handler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if h == "" {
			w.WriteHeader(http.StatusUnauthorized)
			log.Printf("[TO]\t%s Unauthorized\n", r.RemoteAddr)
			return
		}
		u, err := o.authorize(r.Context(), h)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), tenantsKey, u.groups)
		// TODO(adphi): move admins context injection
		ctx = context.WithValue(ctx, adminsKey, o.config.Admins)
		handler.ServeHTTP(w, r.WithContext(ctx))
	}
}

// authorize verifies a bearer token and pulls user information form the claims.
func (o *oidcProvider) authorize(ctx context.Context, bearerToken string) (*user, error) {
	idToken, err := o.verifier.Verify(ctx, bearerToken)
	if err != nil {
		return nil, fmt.Errorf("could not verify bearer token: %v", err)
	}
	// Extract custom claims.
	var claims struct {
		Email    string   `json:"email"`
		Verified bool     `json:"email_verified"`
		Groups   []string `json:"groups"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to parse claims: %v", err)
	}
	if !claims.Verified {
		return nil, fmt.Errorf("email (%q) in returned claims was not verified", claims.Email)
	}
	return &user{claims.Email, claims.Groups}, nil
}

type user struct {
	email  string
	groups []string
}
