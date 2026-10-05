package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	raff "github.com/rafftechnologies/raff-go"
)

func dataSourceDatabase() *schema.Resource {
	return &schema.Resource{
		Description: "Reads an existing managed database by its short `database_id` or by `name`.",
		ReadContext: dataSourceDatabaseRead,
		Schema: map[string]*schema.Schema{
			"database_id":           {Type: schema.TypeString, Optional: true, Computed: true, ExactlyOneOf: []string{"database_id", "name"}, Description: "Short database ID."},
			"name":                  {Type: schema.TypeString, Optional: true, Computed: true, ExactlyOneOf: []string{"database_id", "name"}, Description: "Database name."},
			"engine":                {Type: schema.TypeString, Computed: true},
			"engine_version":        {Type: schema.TypeString, Computed: true},
			"status":                {Type: schema.TypeString, Computed: true},
			"plan_id":               {Type: schema.TypeInt, Computed: true},
			"storage_gb":            {Type: schema.TypeInt, Computed: true},
			"is_free":               {Type: schema.TypeBool, Computed: true},
			"host":                  {Type: schema.TypeString, Computed: true},
			"port":                  {Type: schema.TypeInt, Computed: true},
			"public_host":           {Type: schema.TypeString, Computed: true},
			"public_port":           {Type: schema.TypeInt, Computed: true},
			"username":              {Type: schema.TypeString, Computed: true},
			"password":              {Type: schema.TypeString, Computed: true, Sensitive: true},
			"database_name":         {Type: schema.TypeString, Computed: true},
			"connection_uri":        {Type: schema.TypeString, Computed: true, Sensitive: true},
			"public_connection_uri": {Type: schema.TypeString, Computed: true, Sensitive: true},
		},
	}
}

func dataSourceDatabaseRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	id := d.Get("database_id").(string)
	if id == "" {
		name := d.Get("name").(string)
		dbs, _, err := client.Databases.List(ctx, nil)
		if err != nil {
			return diag.FromErr(err)
		}
		for _, db := range dbs {
			if raff.StringValue(db.Name) == name {
				id = raff.StringValue(db.DatabaseID)
				break
			}
		}
		if id == "" {
			return diag.FromErr(fmt.Errorf("no database named %q", name))
		}
	}
	db, _, err := client.Databases.Get(ctx, id)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(raff.StringValue(db.DatabaseID))
	d.Set("database_id", raff.StringValue(db.DatabaseID))
	d.Set("name", raff.StringValue(db.Name))
	if db.Engine != nil {
		d.Set("engine", string(*db.Engine))
	}
	d.Set("engine_version", raff.StringValue(db.EngineVersion))
	if db.Status != nil {
		d.Set("status", string(*db.Status))
	}
	d.Set("plan_id", raff.IntValue(db.PlanID))
	d.Set("storage_gb", raff.IntValue(db.StorageGb))
	d.Set("is_free", raff.BoolValue(db.IsFree))
	d.Set("host", raff.StringValue(db.ConnectionHost))
	d.Set("port", raff.IntValue(db.ConnectionPort))
	d.Set("public_host", raff.StringValue(db.PublicDNSHostname))
	d.Set("public_port", raff.IntValue(db.PublicPort))
	conn, _, err := client.Databases.Connection(ctx, d.Id(), true)
	if err != nil {
		return diag.FromErr(err)
	}
	d.Set("username", raff.StringValue(conn.Username))
	d.Set("password", raff.StringValue(conn.Password))
	d.Set("database_name", raff.StringValue(conn.DatabaseName))
	d.Set("connection_uri", raff.StringValue(conn.ConnectionURI))
	d.Set("public_connection_uri", raff.StringValue(conn.PublicConnectionURI))
	return nil
}

func dataSourceDatabasePlans() *schema.Resource {
	return &schema.Resource{
		Description: "Managed database plans with pricing. Use a plan `id` as `plan_id` on `raff_database`.",
		ReadContext: dataSourceDatabasePlansRead,
		Schema: map[string]*schema.Schema{
			"engine": {Type: schema.TypeString, Optional: true, Description: "Only plans of this engine."},
			"plans": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id":              {Type: schema.TypeInt, Computed: true},
						"engine":          {Type: schema.TypeString, Computed: true},
						"name":            {Type: schema.TypeString, Computed: true},
						"vcpu":            {Type: schema.TypeInt, Computed: true},
						"memory_gib":      {Type: schema.TypeInt, Computed: true},
						"storage_gib":     {Type: schema.TypeInt, Computed: true},
						"is_free_tier":    {Type: schema.TypeBool, Computed: true},
						"max_connections": {Type: schema.TypeInt, Computed: true},
						"price_per_month": {Type: schema.TypeFloat, Computed: true},
						"price_per_hour":  {Type: schema.TypeFloat, Computed: true},
					},
				},
			},
		},
	}
}

func dataSourceDatabasePlansRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	engine := d.Get("engine").(string)
	plans, _, err := client.Databases.ListPlans(ctx, raff.DatabaseEngine(engine))
	if err != nil {
		return diag.FromErr(err)
	}
	items := make([]any, 0, len(plans.Plans))
	for _, p := range plans.Plans {
		items = append(items, map[string]any{
			"id":              p.ID,
			"engine":          string(p.Engine),
			"name":            p.Name,
			"vcpu":            p.Vcpu,
			"memory_gib":      p.MemoryGib,
			"storage_gib":     p.StorageGib,
			"is_free_tier":    raff.BoolValue(p.IsFreeTier),
			"max_connections": raff.IntValue(p.MaxConnections),
			"price_per_month": float64(p.PricePerMonth),
			"price_per_hour":  float64(p.PricePerHour),
		})
	}
	d.SetId("database-plans-" + engine)
	d.Set("plans", items)
	return nil
}

func dataSourceDatabaseEngines() *schema.Resource {
	return &schema.Resource{
		Description: "Managed database engines and their versions.",
		ReadContext: dataSourceDatabaseEnginesRead,
		Schema: map[string]*schema.Schema{
			"engines": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"engine":       {Type: schema.TypeString, Computed: true},
						"display_name": {Type: schema.TypeString, Computed: true},
						"available":    {Type: schema.TypeBool, Computed: true},
						"versions":     {Type: schema.TypeList, Computed: true, Elem: &schema.Schema{Type: schema.TypeString}},
					},
				},
			},
		},
	}
}

func dataSourceDatabaseEnginesRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	engines, _, err := client.Databases.ListEngines(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	items := make([]any, 0, len(engines))
	for _, e := range engines {
		engine := ""
		if e.Engine != nil {
			engine = string(*e.Engine)
		}
		versions := []string{}
		if e.Versions != nil {
			versions = *e.Versions
		}
		items = append(items, map[string]any{
			"engine":       engine,
			"display_name": raff.StringValue(e.DisplayName),
			"available":    raff.BoolValue(e.Available),
			"versions":     versions,
		})
	}
	d.SetId("database-engines")
	d.Set("engines", items)
	return nil
}
