package provider

import (
	"context"
	"fmt"
	"net"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	raff "github.com/rafftechnologies/raff-go"
)

func resourceDatabase() *schema.Resource {
	return &schema.Resource{
		Description: "Manages a Raff managed database (PostgreSQL, MySQL, Valkey, ClickHouse or Kafka). Without `plan_id` the database is created on the engine's free plan (one per account).",

		CreateContext: resourceDatabaseCreate,
		ReadContext:   resourceDatabaseRead,
		UpdateContext: resourceDatabaseUpdate,
		DeleteContext: resourceDatabaseDelete,
		CustomizeDiff: resourceDatabaseCustomizeDiff,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(20 * time.Minute),
			Update: schema.DefaultTimeout(30 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Display name, unique within the account. Renames in place; hostnames do not change.",
			},
			"engine": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "postgres",
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{"postgres", "mysql", "valkey", "clickhouse", "kafka"}, false),
				Description:  "Engine: `postgres` (default), `mysql`, `valkey`, `clickhouse` or `kafka`.",
			},
			"engine_version": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Engine major version from the `raff_database_engines` data source. Defaults to the latest.",
			},
			"plan_id": {
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Description: "Plan ID from the `raff_database_plans` data source. Defaults to the engine's free plan. Changing it resizes in place (same engine; a free database cannot move to another plan this way).",
			},
			"storage_gb": {
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Description: "Storage in GB. Defaults to the plan's included storage. Grows in place; it cannot shrink.",
			},
			"ha_enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "High availability: a standby that takes over automatically on failure. Toggles in place.",
			},
			"replica_count": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     0,
				Description: "Read replicas (PostgreSQL only). Changes in place.",
			},
			"vpc_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "VPC ID for the private endpoint. Setting, changing or removing it connects or disconnects the database in place.",
			},
			"public_access": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Public access through `public_host`. Toggles in place.",
			},
			"public_allowlist": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    20,
				Elem:        &schema.Schema{Type: schema.TypeString, ValidateFunc: validateAllowlistEntry},
				Description: "Source IPv4 CIDRs allowed on the public endpoint, e.g. `203.0.113.0/24` or `198.51.100.7/32` (at most 20). Empty allows all sources; `0.0.0.0/0` is not accepted.",
			},
			"extensions": {
				Type:        schema.TypeSet,
				Optional:    true,
				Set:         schema.HashString,
				Elem:        &schema.Schema{Type: schema.TypeString, ValidateFunc: validation.StringInSlice(databaseCreateExtensions, false)},
				Description: "PostgreSQL only. Extensions to turn on: `vector`, `pg_trgm`, `pg_stat_statements`, `hstore`, `uuid-ossp`, `citext`, `ltree`, `pgcrypto` or `unaccent`. At create they are on before the database is running; adding one later turns it on in place. Removing one only stops Terraform managing it, it is never dropped.",
			},
			"valkey_mode": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"queue", "cache"}, false),
				Description:  "Valkey only, set at create. `queue` (default) keeps every key and refuses writes when memory is full; `cache` drops the least recently used keys. It cannot change on a running database yet, so a change is refused instead of replacing the database. Not read back from the API.",
			},

			// Computed
			"database_id":           {Type: schema.TypeString, Computed: true, Description: "Short database ID used in hostnames and the API."},
			"status":                {Type: schema.TypeString, Computed: true},
			"is_free":               {Type: schema.TypeBool, Computed: true, Description: "True on the free plan. Free databases pause after 7 days without connections."},
			"host":                  {Type: schema.TypeString, Computed: true, Description: "Private hostname, reachable inside the VPC."},
			"port":                  {Type: schema.TypeInt, Computed: true, Description: "Private port."},
			"public_host":           {Type: schema.TypeString, Computed: true, Description: "Public hostname (empty while public access is off)."},
			"public_port":           {Type: schema.TypeInt, Computed: true, Description: "Public port. For PostgreSQL this is the pooled endpoint and `public_port + 1` is direct; the standard ports 6543 and 5432 also work on `public_host`."},
			"username":              {Type: schema.TypeString, Computed: true},
			"password":              {Type: schema.TypeString, Computed: true, Sensitive: true},
			"database_name":         {Type: schema.TypeString, Computed: true},
			"connection_uri":        {Type: schema.TypeString, Computed: true, Sensitive: true, Description: "Private connection URI including the password (reachable inside the VPC)."},
			"public_connection_uri": {Type: schema.TypeString, Computed: true, Sensitive: true, Description: "Public connection URI including the password (empty while public access is off)."},
			"ca_cert":               {Type: schema.TypeString, Computed: true, Description: "CA that signs the database's certificate. Kafka clients must trust it; PostgreSQL can use it with sslmode=verify-full."},
			"monthly_price":         {Type: schema.TypeFloat, Computed: true},
			"project_id":            {Type: schema.TypeString, Computed: true},
			"created_at":            {Type: schema.TypeString, Computed: true},
		},
	}
}

// databaseCreateExtensions is what the API accepts in create's extensions.
var databaseCreateExtensions = []string{
	"vector", "pg_trgm", "pg_stat_statements", "hstore",
	"uuid-ossp", "citext", "ltree", "pgcrypto", "unaccent",
}

// valkeyModePolicy maps valkey_mode to the engine_config value the API accepts.
var valkeyModePolicy = map[string]string{
	"queue": "noeviction",
	"cache": "allkeys-lru",
}

func databaseExtensions(d resourceGetter) []string {
	out := []string{}
	if set, ok := d.Get("extensions").(*schema.Set); ok {
		for _, v := range set.List() {
			out = append(out, v.(string))
		}
	}
	return out
}

type resourceGetter interface{ Get(string) any }

// resourceDatabaseCustomizeDiff refuses settings the engine does not have, and
// a valkey_mode change on a running database (the API cannot change it yet,
// and replacing a Valkey would lose its keys).
func resourceDatabaseCustomizeDiff(_ context.Context, d *schema.ResourceDiff, _ any) error {
	engine := d.Get("engine").(string)
	if len(databaseExtensions(d)) > 0 && engine != "postgres" {
		return fmt.Errorf("extensions are for PostgreSQL only, not %s", engine)
	}
	if d.Get("valkey_mode").(string) != "" && engine != "valkey" {
		return fmt.Errorf("valkey_mode is for Valkey only, not %s", engine)
	}
	if d.Id() != "" && d.HasChange("valkey_mode") {
		if old, _ := d.GetChange("valkey_mode"); old.(string) != "" {
			return fmt.Errorf("valkey_mode is set at create and cannot change on a running database yet")
		}
	}
	return nil
}

// databaseFreePlanID returns the engine's free-tier plan.
func databaseFreePlanID(ctx context.Context, client *raff.Client, engine raff.DatabaseEngine) (int, error) {
	plans, _, err := client.Databases.ListPlans(ctx, engine)
	if err != nil {
		return 0, err
	}
	for _, p := range plans.Plans {
		if raff.BoolValue(p.IsFreeTier) {
			return p.ID, nil
		}
	}
	return 0, fmt.Errorf("engine %s has no free plan: set plan_id", engine)
}

func resourceDatabaseCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)

	engine := raff.DatabaseEngine(d.Get("engine").(string))
	planID := d.Get("plan_id").(int)
	if planID == 0 {
		id, err := databaseFreePlanID(ctx, client, engine)
		if err != nil {
			return diag.FromErr(err)
		}
		planID = id
	}
	req := &raff.CreateDatabaseRequest{Name: d.Get("name").(string), Engine: &engine, PlanID: planID}
	if v, ok := d.GetOk("engine_version"); ok {
		s := v.(string)
		req.EngineVersion = &s
	}
	if v, ok := d.GetOk("storage_gb"); ok {
		n := v.(int)
		req.StorageGb = &n
	}
	if d.Get("ha_enabled").(bool) {
		req.HaEnabled = raff.Bool(true)
	}
	if n := d.Get("replica_count").(int); n > 0 {
		req.ReplicaCount = &n
	}
	if v, ok := d.GetOk("vpc_id"); ok {
		id, err := uuid.Parse(v.(string))
		if err != nil {
			return diag.Errorf("invalid vpc_id: %s", err)
		}
		req.VpcID = &id
	}
	if exts := databaseExtensions(d); len(exts) > 0 {
		req.Extensions = &exts
	}
	if mode := d.Get("valkey_mode").(string); mode != "" {
		req.EngineConfig = &map[string]string{"maxmemory-policy": valkeyModePolicy[mode]}
	}

	db, _, err := client.Databases.Create(ctx, req)
	if err != nil {
		return diag.FromErr(err)
	}
	id := raff.StringValue(db.DatabaseID)
	d.SetId(id)

	waitCtx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutCreate))
	defer cancel()
	if _, err := client.Databases.WaitForStatus(waitCtx, id, raff.DatabaseStatusRunning); err != nil {
		return diag.Errorf("database %s did not reach running: %s", id, err)
	}

	// Public access goes through its own endpoint so the allowlist is applied
	// with it.
	if d.Get("public_access").(bool) {
		if _, _, err := client.Databases.SetPublicAccess(ctx, id, true, databaseAllowlist(d)); err != nil {
			return diag.FromErr(err)
		}
		waitForDatabasePublic(waitCtx, client, id)
	}
	return resourceDatabaseRead(ctx, d, meta)
}

// waitForDatabasePublic holds the apply until the public address answers, so
// connection_uri works for whatever runs next (the gateway routes a database
// 10 to 20 seconds after it is running). It does not fail the apply: an
// allowlist can keep the machine running Terraform out.
func waitForDatabasePublic(ctx context.Context, client *raff.Client, id string) {
	conn, _, err := client.Databases.Connection(ctx, id, false)
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	_ = raff.WaitForPublicEndpoint(ctx, conn)
}

// validateAllowlistEntry accepts what the API stores unchanged: an IPv4 CIDR
// written with its network address and a prefix above 0. Anything the API
// would rewrite (a bare IP, 203.0.113.5/24) shows as drift on every plan.
func validateAllowlistEntry(v any, key string) ([]string, []error) {
	s, _ := v.(string)
	ip, ipnet, err := net.ParseCIDR(s)
	switch {
	case err != nil || ip.To4() == nil:
		return nil, []error{fmt.Errorf("%s: %q is not an IPv4 CIDR (write a single address as 198.51.100.7/32)", key, s)}
	case ipnet.String() != s:
		return nil, []error{fmt.Errorf("%s: write %q as %q", key, s, ipnet.String())}
	}
	if ones, _ := ipnet.Mask.Size(); ones == 0 {
		return nil, []error{fmt.Errorf("%s: 0.0.0.0/0 is not accepted; leave public_allowlist empty to allow all sources", key)}
	}
	return nil, nil
}

func databaseAllowlist(d *schema.ResourceData) []string {
	list := []string{}
	for _, v := range d.Get("public_allowlist").([]any) {
		if s, ok := v.(string); ok {
			list = append(list, s)
		}
	}
	return list
}

func resourceDatabaseRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	db, _, err := client.Databases.Get(ctx, d.Id())
	if err != nil {
		if errResp, ok := err.(*raff.ErrorResponse); ok && errResp.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	if db.Status != nil && *db.Status == raff.DatabaseStatusDeleted {
		d.SetId("")
		return nil
	}

	// Keep the short id as the resource id (an import by UUID converges here).
	d.SetId(raff.StringValue(db.DatabaseID))
	d.Set("database_id", raff.StringValue(db.DatabaseID))
	d.Set("name", raff.StringValue(db.Name))
	if db.Engine != nil {
		d.Set("engine", string(*db.Engine))
	}
	d.Set("engine_version", raff.StringValue(db.EngineVersion))
	d.Set("plan_id", raff.IntValue(db.PlanID))
	d.Set("storage_gb", raff.IntValue(db.StorageGb))
	d.Set("ha_enabled", raff.BoolValue(db.HaEnabled))
	d.Set("replica_count", raff.IntValue(db.ReplicaCount))
	d.Set("vpc_id", raff.StringValue(db.VpcID))
	d.Set("public_access", raff.BoolValue(db.PublicAccess))
	if db.PublicAllowlist != nil {
		d.Set("public_allowlist", *db.PublicAllowlist)
	} else {
		d.Set("public_allowlist", []string{})
	}
	if db.Status != nil {
		d.Set("status", string(*db.Status))
	}
	d.Set("is_free", raff.BoolValue(db.IsFree))
	d.Set("host", raff.StringValue(db.ConnectionHost))
	d.Set("port", raff.IntValue(db.ConnectionPort))
	d.Set("public_host", raff.StringValue(db.PublicDNSHostname))
	d.Set("public_port", raff.IntValue(db.PublicPort))
	if db.MonthlyPrice != nil {
		d.Set("monthly_price", float64(*db.MonthlyPrice))
	}
	if db.ProjectID != nil {
		d.Set("project_id", db.ProjectID.String())
	}
	if db.CreatedAt != nil {
		d.Set("created_at", db.CreatedAt.Format(time.RFC3339))
	}

	// Only the configured extensions are tracked (others may be turned on in
	// the dashboard); one turned off elsewhere shows as a diff.
	if want := databaseExtensions(d); len(want) > 0 && db.Status != nil && *db.Status == raff.DatabaseStatusRunning {
		if exts, _, err := client.Databases.ListExtensions(ctx, d.Id()); err == nil {
			on := []string{}
			for _, e := range exts {
				name := raff.StringValue(e.Name)
				if raff.BoolValue(e.Installed) && slices.Contains(want, name) {
					on = append(on, name)
				}
			}
			d.Set("extensions", on)
		}
	}

	if conn, _, err := client.Databases.Connection(ctx, d.Id(), true); err == nil {
		d.Set("username", raff.StringValue(conn.Username))
		d.Set("password", raff.StringValue(conn.Password))
		d.Set("database_name", raff.StringValue(conn.DatabaseName))
		d.Set("connection_uri", raff.StringValue(conn.ConnectionURI))
		d.Set("public_connection_uri", raff.StringValue(conn.PublicConnectionURI))
		d.Set("ca_cert", raff.StringValue(conn.CaCert))
	}
	return nil
}

func resourceDatabaseUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	id := d.Id()

	if d.HasChange("name") {
		if _, _, err := client.Databases.Rename(ctx, id, d.Get("name").(string)); err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChanges("plan_id", "storage_gb", "ha_enabled", "replica_count") {
		req := &raff.ScaleDatabaseRequest{ReplicaCount: raff.Int(d.Get("replica_count").(int))}
		if d.HasChange("plan_id") {
			req.PlanID = raff.Int(d.Get("plan_id").(int))
		}
		if d.HasChange("storage_gb") {
			old, current := d.GetChange("storage_gb")
			if current.(int) < old.(int) {
				return diag.Errorf("storage_gb cannot shrink (%d to %d)", old.(int), current.(int))
			}
			req.StorageGb = raff.Int(current.(int))
		}
		if d.HasChange("ha_enabled") {
			req.SetHa = raff.Bool(true)
			req.HasHa = raff.Bool(d.Get("ha_enabled").(bool))
		}
		if _, _, err := client.Databases.Scale(ctx, id, req); err != nil {
			return diag.FromErr(err)
		}
		waitCtx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutUpdate))
		defer cancel()
		if _, err := client.Databases.WaitForStatus(waitCtx, id, raff.DatabaseStatusRunning); err != nil {
			return diag.Errorf("database %s did not return to running after the resize: %s", id, err)
		}
	}

	if d.HasChange("vpc_id") {
		old, current := d.GetChange("vpc_id")
		if old.(string) != "" {
			if _, _, err := client.Databases.DisconnectVPC(ctx, id); err != nil {
				return diag.FromErr(err)
			}
		}
		if current.(string) != "" {
			if _, _, err := client.Databases.ConnectVPC(ctx, id, current.(string)); err != nil {
				return diag.FromErr(err)
			}
		}
	}

	// Added extensions are turned on; removed ones are only no longer
	// managed (dropping one could fail or remove data that uses it).
	if d.HasChange("extensions") {
		old, current := d.GetChange("extensions")
		for _, v := range current.(*schema.Set).Difference(old.(*schema.Set)).List() {
			if _, err := client.Databases.SetExtension(ctx, id, v.(string), true); err != nil {
				return diag.Errorf("turn on extension %s: %s", v.(string), err)
			}
		}
	}

	if d.HasChanges("public_access", "public_allowlist") {
		enabled := d.Get("public_access").(bool)
		var list []string
		if enabled {
			list = databaseAllowlist(d)
		}
		if _, _, err := client.Databases.SetPublicAccess(ctx, id, enabled, list); err != nil {
			return diag.FromErr(err)
		}
		if enabled {
			waitForDatabasePublic(ctx, client, id)
		}
	}

	return resourceDatabaseRead(ctx, d, meta)
}

func resourceDatabaseDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	if _, err := client.Databases.Delete(ctx, d.Id()); err != nil {
		if errResp, ok := err.(*raff.ErrorResponse); ok && errResp.StatusCode == 404 {
			return nil
		}
		return diag.FromErr(err)
	}
	waitCtx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutDelete))
	defer cancel()
	if err := client.Databases.WaitForDeleted(waitCtx, d.Id()); err != nil {
		return diag.Errorf("database %s deletion did not complete: %s", d.Id(), err)
	}
	return nil
}
