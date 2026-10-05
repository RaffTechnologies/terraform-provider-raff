package provider

import "testing"

func TestDatabaseUserRole(t *testing.T) {
	for access, want := range map[string]string{"read only": "readonly", "read/write": "readwrite", "read/write + DDL": "readwrite"} {
		if got := databaseUserRole(access); got != want {
			t.Errorf("%q: got %s, want %s", access, got, want)
		}
	}
}
