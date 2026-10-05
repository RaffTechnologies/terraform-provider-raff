package provider

import "testing"

func TestDatabaseUserRole(t *testing.T) {
	for access, want := range map[string]string{"read only": "readonly", "read/write": "readwrite", "read/write + DDL": "readwrite"} {
		if got := databaseUserRole(access); got != want {
			t.Errorf("%q: got %s, want %s", access, got, want)
		}
	}
}

func TestValidateAllowlistEntry(t *testing.T) {
	for entry, ok := range map[string]bool{
		"203.0.113.0/24": true, "198.51.100.7/32": true,
		"0.0.0.0/0": false, "198.51.100.7": false, "203.0.113.5/24": false, "2001:db8::/32": false, "x": false,
	} {
		if _, errs := validateAllowlistEntry(entry, "public_allowlist.0"); (len(errs) == 0) != ok {
			t.Errorf("%s: errors %v, want ok=%v", entry, errs, ok)
		}
	}
}

func TestValidateDatabaseUserName(t *testing.T) {
	for name, ok := range map[string]bool{"reporting": true, "app2": true, "app_rw": false, "fnb_x": false, "Bad": false, "ab": false} {
		if _, errs := validateDatabaseUserName(name, "name"); (len(errs) == 0) != ok {
			t.Errorf("%s: errors %v, want ok=%v", name, errs, ok)
		}
	}
}
