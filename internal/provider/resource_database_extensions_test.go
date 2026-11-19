package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// The provider checks extension names before the API does. Its list must
// hold every name the API takes at create: postgis and pg_cron joined it on
// 2026-11-19 (PostgreSQL databases created from that day).
func TestDatabaseExtensionsAcceptWhatTheAPITakes(t *testing.T) {
	elem, ok := resourceDatabase().Schema["extensions"].Elem.(*schema.Schema)
	if !ok || elem.ValidateFunc == nil {
		t.Fatal("extensions has no name check")
	}
	for _, name := range []string{
		"vector", "postgis", "pg_cron", "pg_trgm", "pg_stat_statements", "hstore",
		"uuid-ossp", "citext", "ltree", "pgcrypto", "unaccent",
	} {
		if _, errs := elem.ValidateFunc(name, "extensions"); len(errs) > 0 {
			t.Errorf("%s must be accepted: %v", name, errs)
		}
	}
	if _, errs := elem.ValidateFunc("plpython3u", "extensions"); len(errs) == 0 {
		t.Error("a name outside the list must be refused")
	}
}
