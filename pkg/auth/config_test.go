package auth

import (
	"reflect"
	"testing"
)

func TestParseConfig(t *testing.T) {
	configInvalidLocation := "../../configs/no.config.yaml"
	configInvalidConfigFileLocation := "../../configs/bad.yaml"
	configSampleLocation := "../../configs/sample.yaml"
	configMultipleUserLocation := "../../configs/multiple.user.yaml"
	expectedSampleAuth := Authn{
		StaticUsers: []User{
			{
				Username: "Happy",
				Password: "Prometheus",
				Tenants:  []string{"default"},
			}, {
				Username: "Sad",
				Password: "Prometheus",
				Tenants:  []string{"kube-system"},
			},
		},
	}
	expectedMultipleUserAuth := Authn{
		Admins: []string{"admin"},
		StaticUsers: []User{
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
			{
				Username: "admin",
				Password: "admin",
				Tenants:  []string{"admin"},
			},
		},
	}
	type args struct {
		location string
	}
	tests := []struct {
		name    string
		args    args
		want    *Authn
		wantErr bool
	}{
		{
			"Basic",
			args{
				configSampleLocation,
			},
			&expectedSampleAuth,
			false,
		}, {
			"Multiples users",
			args{
				configMultipleUserLocation,
			},
			&expectedMultipleUserAuth,
			false,
		}, {
			"Invalid location",
			args{
				configInvalidLocation,
			},
			nil,
			true,
		}, {
			"Invalid yaml file",
			args{
				configInvalidConfigFileLocation,
			},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseConfig(tt.args.location)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}
