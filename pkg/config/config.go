package config

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// Authn Contains a list of users
type Authn struct {
	StaticUsers []User   `yaml:"static_users"`
	OIDC        *OIDC    `yaml:"oidc"`
	Admins      []string `yaml:"admins"`
}

// User Identifies a user including the tenant
type User struct {
	Username string   `yaml:"username"`
	Password string   `yaml:"password"`
	Tenants  []string `yaml:"tenants"`
}

type OIDC struct {
	IssuerURL string `yaml:"issuer_url"`
	ClientID  string `yaml:"client_id"`
}

// Parse read a configuration file in the path `location` and returns an Authn object
func Parse(location string) (*Authn, error) {
	data, err := ioutil.ReadFile(location)
	if err != nil {
		return nil, err
	}
	authn := &Authn{}
	err = yaml.Unmarshal(data, authn)
	if err != nil {
		return nil, err
	}
	return authn, nil
}
