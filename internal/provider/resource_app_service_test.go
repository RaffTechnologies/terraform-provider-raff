package provider

import (
	"reflect"
	"testing"

	raff "github.com/rafftechnologies/raff-go"
)

func TestAppDeploysOnCreate(t *testing.T) {
	for _, tc := range []struct {
		source, builder string
		want            bool
	}{
		{"image", "", true}, {"git", "buildpacks", true}, {"", "image", true},
		{"console", "dockerfile", false}, {"template", "buildpacks", false},
	} {
		if got := appDeploysOnCreate(tc.source, tc.builder); got != tc.want {
			t.Errorf("appDeploysOnCreate(%q, %q) = %v, want %v", tc.source, tc.builder, got, tc.want)
		}
	}
}

// Only the keys in the configuration are tracked; a database link's
// variables and ones added outside Terraform never show as drift.
func TestManagedAppEnv(t *testing.T) {
	vars := []raff.AppEnvVar{
		{Key: "GREETING", Value: "hi"},
		{Key: "API_KEY", Value: "s3cret", IsSecret: true},
		{Key: "DATABASE_URL", Value: "postgres://x", IsSecret: true, IsSystem: true},
		{Key: "ADDED_IN_DASHBOARD", Value: "1"},
	}
	env, secret := managedAppEnv(vars, map[string]any{"GREETING": "old"}, map[string]any{"API_KEY": "old", "DATABASE_URL": "x"})
	if want := map[string]string{"GREETING": "hi"}; !reflect.DeepEqual(env, want) {
		t.Errorf("env = %v, want %v", env, want)
	}
	if want := map[string]string{"API_KEY": "s3cret"}; !reflect.DeepEqual(secret, want) {
		t.Errorf("secret_env = %v, want %v", secret, want)
	}
}

func TestSplitAppChildID(t *testing.T) {
	if s, c, err := splitAppChildID("svc-uuid/child-uuid"); err != nil || s != "svc-uuid" || c != "child-uuid" {
		t.Errorf("got %q %q %v", s, c, err)
	}
	for _, bad := range []string{"", "nochild", "/x", "x/"} {
		if _, _, err := splitAppChildID(bad); err == nil {
			t.Errorf("%q: want an error", bad)
		}
	}
}
