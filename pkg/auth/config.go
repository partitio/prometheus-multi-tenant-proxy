package auth

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// Authn Contains a list of users
type Authn struct {
	StaticUsers []User   `yaml:"static_users"`
	Admins      []string `yaml:"admins"`
}

// User Identifies a user including the tenant
type User struct {
	Username string   `yaml:"username"`
	Password string   `yaml:"password"`
	Tenants  []string `yaml:"tenants"`
}

// ParseConfig read a configuration file in the path `location` and returns an Authn object
func ParseConfig(location string) (*Authn, error) {
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
