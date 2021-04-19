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
	if config.OIDC.CookieName == "" {
		config.OIDC.CookieName = "id_token"
	}
	// Initialize a provider by specifying dex's issuer URL.
	provider, err := oidc.NewProvider(context.Background(), config.OIDC.IssuerURL)
	if err != nil {
		return nil, err
	}
	c := &oidc.Config{ClientID: config.OIDC.ClientID}
	if config.OIDC.ClientID == "" {
		c.SkipClientIDCheck = true
	}
	// Create an ID token parser, but only trust ID tokens issued to "example-app"
	idTokenVerifier := provider.Verifier(c)
	return &oidcProvider{config: config, verifier: idTokenVerifier}, nil
}

type oidcProvider struct {
	config   *config.Authn
	verifier *oidc.IDTokenVerifier
}

func (o *oidcProvider) Authenticate(handler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var tk string
		if tk = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "); tk != "" {
			log.Println("Token source: Header")
		} else if c, err := r.Cookie(o.config.OIDC.CookieName); err == nil {
			tk = c.Value
			log.Println("Token source: Cookie")
		}
		if tk == "" {
			w.WriteHeader(http.StatusUnauthorized)
			log.Printf("[TO]\t%s Unauthorized: no Bearer token\n", r.RemoteAddr)
			return
		}
		u, err := o.authorize(r.Context(), tk)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			log.Printf("[TO]\t%s Unauthorized: %v\n", r.RemoteAddr, err)
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
