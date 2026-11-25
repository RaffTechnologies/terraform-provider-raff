package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	raff "github.com/rafftechnologies/raff-go"
)

const upgradeTestDBID = "a1b2c3d4"

// A version change must never replace the database: replacing deletes its
// data, and the database may have been upgraded outside Terraform.
func TestDatabaseEngineVersionIsNotForceNew(t *testing.T) {
	if resourceDatabase().Schema["engine_version"].ForceNew {
		t.Fatal("engine_version must not be ForceNew: a version change would delete the database")
	}
}

// databaseState is a running database as it sits in state after a read.
func databaseState(engine, version string) *terraform.InstanceState {
	return &terraform.InstanceState{
		ID: upgradeTestDBID,
		Attributes: map[string]string{
			"id":                 upgradeTestDBID,
			"name":               "orders",
			"engine":             engine,
			"engine_version":     version,
			"plan_id":            "3",
			"storage_gb":         "20",
			"ha_enabled":         "false",
			"replica_count":      "0",
			"public_access":      "false",
			"upgrade_versions.#": "1",
			"upgrade_versions.0": "19",
		},
	}
}

func databaseConfig(engine, version string, extra map[string]any) *terraform.ResourceConfig {
	raw := map[string]any{"name": "orders", "engine": engine, "plan_id": 3, "storage_gb": 20}
	if version != "" {
		raw["engine_version"] = version
	}
	for k, v := range extra {
		raw[k] = v
	}
	return terraform.NewResourceConfigRaw(raw)
}

func planDatabase(t *testing.T, state *terraform.InstanceState, cfg *terraform.ResourceConfig) (*terraform.InstanceDiff, error) {
	t.Helper()
	return resourceDatabase().SimpleDiff(context.Background(), state, cfg, nil)
}

// Another engine cannot change version in place, and the plan says so
// instead of replacing the database.
func TestDatabaseVersionChangeRefusedOnOtherEngines(t *testing.T) {
	_, err := planDatabase(t, databaseState("mysql", "8.0"), databaseConfig("mysql", "8.4", nil))
	want := "engine_version cannot change on a running mysql database (8.0 to 8.4). Only PostgreSQL upgrades in place. To move to another version, create a new raff_database on it and move the data."
	if err == nil || err.Error() != want {
		t.Fatalf("error %v, want %q", err, want)
	}
}

// A lower major is refused, with the way out for a database upgraded
// outside Terraform.
func TestDatabaseVersionLoweringRefused(t *testing.T) {
	_, err := planDatabase(t, databaseState("postgres", "19"), databaseConfig("postgres", "18", nil))
	want := `engine_version cannot go from 19 to 18: a database does not move to an older major version in place. If the database was upgraded outside Terraform, set engine_version = "19". To go back to 18, restore a backup from before the upgrade (dashboard or API).`
	if err == nil || err.Error() != want {
		t.Fatalf("error %v, want %q", err, want)
	}
}

// Versions that are not plain numbers are left to the API.
func TestDatabaseVersionNotNumbersLeftToAPI(t *testing.T) {
	if err := checkDatabaseVersionChange("postgres", "18", "latest"); err != nil {
		t.Errorf("got %v, want no refusal", err)
	}
}

// A higher major on PostgreSQL plans an in-place update.
func TestDatabaseVersionRaisePlansInPlace(t *testing.T) {
	diff, err := planDatabase(t, databaseState("postgres", "18"), databaseConfig("postgres", "19", nil))
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if diff == nil || diff.Attributes["engine_version"] == nil || diff.Attributes["engine_version"].New != "19" {
		t.Fatalf("engine_version is not in the plan: %+v", diff)
	}
	if diff.RequiresNew() {
		t.Fatal("raising engine_version must not replace the database")
	}
	if diff.Destroy {
		t.Fatal("raising engine_version must not destroy the database")
	}
}

// Without engine_version in the configuration, a database upgraded in the
// dashboard (state read back as 19) plans no change.
func TestDatabaseUnsetVersionNoDiffAfterOutsideUpgrade(t *testing.T) {
	diff, err := planDatabase(t, databaseState("postgres", "19"), databaseConfig("postgres", "", nil))
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if diff != nil && !diff.Empty() {
		t.Fatalf("want no change, got %+v", diff.Attributes)
	}
}

// fakeDatabaseAPI is a PostgreSQL 18 database. An upgrade either succeeds
// or ends failed with failMessage; a refusal answers 412 with refuseMessage.
type fakeDatabaseAPI struct {
	mu            sync.Mutex
	version       string
	upgrade       string
	refuseMessage string
	failMessage   string
	calls         []string
}

func (f *fakeDatabaseAPI) dbBody() string {
	return `{"success":true,"database":{"database_id":"` + upgradeTestDBID + `","name":"orders","engine":"postgres","engine_version":"` + f.version +
		`","status":"running","plan_id":3,"storage_gb":20,"upgrade_versions":[],"last_version_upgrade":` + f.upgrade + `}}`
}

func (f *fakeDatabaseAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	base := "/api/v1/databases/" + upgradeTestDBID
	w.Header().Set("Content-Type", "application/json")
	switch {
	case r.Method == http.MethodPost && r.URL.Path == base+"/upgrade":
		f.calls = append(f.calls, "upgrade")
		if f.refuseMessage != "" {
			w.WriteHeader(http.StatusPreconditionFailed)
			_, _ = w.Write([]byte(`{"success":false,"error":"` + f.refuseMessage + `"}`))
			return
		}
		if f.failMessage != "" {
			f.upgrade = `{"from_version":"18","to_version":"19","state":"failed","message":"` + f.failMessage + `"}`
		} else {
			f.version = "19"
			f.upgrade = `{"from_version":"18","to_version":"19","state":"analyzing","message":""}`
		}
		_, _ = w.Write([]byte(f.dbBody()))
	case r.Method == http.MethodPost && r.URL.Path == base+"/scale":
		f.calls = append(f.calls, "scale")
		_, _ = w.Write([]byte(f.dbBody()))
	case r.Method == http.MethodGet && r.URL.Path == base:
		_, _ = w.Write([]byte(f.dbBody()))
	default:
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"success":false,"error":"not found"}`))
	}
}

// applyDatabase plans config against a PostgreSQL 18 in state and applies it
// against the fake API, as terraform apply does.
func applyDatabase(t *testing.T, api *fakeDatabaseAPI, cfg *terraform.ResourceConfig) (*terraform.InstanceState, diag.Diagnostics) {
	t.Helper()
	srv := httptest.NewServer(api)
	t.Cleanup(srv.Close)
	client := raff.NewFromToken("test", raff.SetBaseURL(srv.URL), raff.SetProjectID("00000000-0000-0000-0000-000000000001"))

	state := databaseState("postgres", "18")
	diff, err := resourceDatabase().SimpleDiff(context.Background(), state, cfg, client)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	return resourceDatabase().Apply(context.Background(), state, diff, client)
}

func newFakeDatabaseAPI() *fakeDatabaseAPI {
	return &fakeDatabaseAPI{version: "18", upgrade: "null"}
}

// A raised version upgrades, waits, and reads back the new version.
func TestDatabaseUpgradeApplySucceeds(t *testing.T) {
	api := newFakeDatabaseAPI()
	state, diags := applyDatabase(t, api, databaseConfig("postgres", "19", nil))
	if diags.HasError() {
		t.Fatalf("apply: %v", diags)
	}
	if got := strings.Join(api.calls, ","); got != "upgrade" {
		t.Errorf("calls %s, want upgrade", got)
	}
	if state.Attributes["engine_version"] != "19" || state.ID != upgradeTestDBID {
		t.Errorf("state engine_version %q id %q", state.Attributes["engine_version"], state.ID)
	}
}

// An upgrade that did not complete shows the API's sentence, and the state
// keeps the version the database is really on.
func TestDatabaseUpgradeApplyFailedKeepsOldVersion(t *testing.T) {
	const reason = "The upgrade to PostgreSQL 19 could not be completed, so your database is back on PostgreSQL 18, unchanged."
	api := newFakeDatabaseAPI()
	api.failMessage = reason
	state, diags := applyDatabase(t, api, databaseConfig("postgres", "19", map[string]any{"name": "orders-renamed"}))
	if !diags.HasError() {
		t.Fatal("want an error")
	}
	if diags[0].Summary != "The upgrade to PostgreSQL 19 did not complete" || diags[0].Detail != reason {
		t.Errorf("diagnostic %q / %q", diags[0].Summary, diags[0].Detail)
	}
	if state == nil || state.Attributes["engine_version"] != "18" || state.ID != upgradeTestDBID {
		t.Fatalf("state %+v, want engine_version 18", state)
	}
	if state.Attributes["name"] != "orders" {
		t.Errorf("state name %q: the rename did not run", state.Attributes["name"])
	}
	if got := strings.Join(api.calls, ","); got != "upgrade" {
		t.Errorf("calls %s, want only upgrade", got)
	}
}

// A refusal shows the API's message unchanged and changes nothing.
func TestDatabaseUpgradeApplyRefused(t *testing.T) {
	const refusal = "Column public.events.payload uses type abstime, which PostgreSQL 19 cannot carry over."
	api := newFakeDatabaseAPI()
	api.refuseMessage = refusal
	state, diags := applyDatabase(t, api, databaseConfig("postgres", "19", map[string]any{"plan_id": 4}))
	if !diags.HasError() {
		t.Fatal("want an error")
	}
	if diags[0].Detail != refusal {
		t.Errorf("detail %q, want the API's message", diags[0].Detail)
	}
	if want := "The upgrade to PostgreSQL 19 was refused. The database is unchanged on PostgreSQL 18"; diags[0].Summary != want {
		t.Errorf("summary %q", diags[0].Summary)
	}
	if state == nil || state.Attributes["engine_version"] != "18" || state.Attributes["plan_id"] != "3" {
		t.Fatalf("state %+v, want engine_version 18 and plan_id 3", state)
	}
	if got := strings.Join(api.calls, ","); got != "upgrade" {
		t.Errorf("calls %s, want only upgrade", got)
	}
}

// A resize in the same apply runs after the upgrade has finished.
func TestDatabaseUpgradeRunsBeforeScale(t *testing.T) {
	api := newFakeDatabaseAPI()
	_, diags := applyDatabase(t, api, databaseConfig("postgres", "19", map[string]any{"plan_id": 4}))
	if diags.HasError() {
		t.Fatalf("apply: %v", diags)
	}
	if got := strings.Join(api.calls, ","); got != "upgrade,scale" {
		t.Errorf("calls %s, want upgrade,scale", got)
	}
}
