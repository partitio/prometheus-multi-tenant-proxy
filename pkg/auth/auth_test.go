package auth

import (
	"strings"
	"testing"

	"github.com/k8spin/prometheus-multi-tenant-proxy/pkg/config"
)

func Test_isAuthorized(t *testing.T) {
	authConfig := config.Authn{
		StaticUsers: []config.User{
			{
				Username: "User-a",
				Password: "pass-a",
				Tenants:  []string{"tenant-a"},
			},
			{
				Username: "User-b",
				Password: "pass-b",
				Tenants:  []string{"tenant-b"},
			},
		},
	}
	type args struct {
		user       string
		pass       string
		authConfig *config.Authn
	}
	tests := []struct {
		name  string
		args  args
		want  bool
		want1 []string
	}{
		{
			"Valid User",
			args{
				"User-a",
				"pass-a",
				&authConfig,
			},
			true,
			[]string{"tenant-a"},
		}, {
			"Invalid User",
			args{
				"invalid",
				"pass-a",
				&authConfig,
			},
			false,
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := isAuthorized(tt.args.user, tt.args.pass, tt.args.authConfig)
			if got != tt.want {
				t.Errorf("isAuthorized() got = %v, want %v", got, tt.want)
			}
			if strings.Join(got1, ",") != strings.Join(tt.want1, ",") {
				t.Errorf("isAuthorized() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
