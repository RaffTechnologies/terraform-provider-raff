package provider

import (
	"context"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	raff "github.com/rafftechnologies/raff-go"
)

func backupComputedSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"id":           {Type: schema.TypeString, Computed: true},
		"name":         {Type: schema.TypeString, Computed: true},
		"vm_id":        {Type: schema.TypeString, Computed: true},
		"status":       {Type: schema.TypeString, Computed: true},
		"storage_size": {Type: schema.TypeInt, Computed: true},
		"region":       {Type: schema.TypeString, Computed: true},
		"expire_date":  {Type: schema.TypeString, Computed: true},
		"account_id":   {Type: schema.TypeString, Computed: true},
		"project_id":   {Type: schema.TypeString, Computed: true},
		"created_at":   {Type: schema.TypeString, Computed: true},
	}
}

func dataSourceBackup() *schema.Resource {
	s := backupComputedSchema()
	s["id"] = &schema.Schema{Type: schema.TypeString, Required: true}
	return &schema.Resource{
		Description: "Reads a single backup by ID.",
		ReadContext: dataSourceBackupRead,
		Schema:      s,
	}
}

func dataSourceBackupRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	b, _, err := client.Backups.Get(ctx, d.Get("id").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(b.ID.String())
	for k, v := range backupToMap(b) {
		d.Set(k, v)
	}
	return nil
}

func dataSourceBackups() *schema.Resource {
	return &schema.Resource{
		Description: "Lists backups in the current project.",
		ReadContext: dataSourceBackupsRead,
		Schema: map[string]*schema.Schema{
			"vm_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Filter by source VM UUID.",
			},
			"backups": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Resource{Schema: backupComputedSchema()},
			},
		},
	}
}

func dataSourceBackupsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	opts := &raff.BackupListOptions{}
	// Apply server-side vm_id filter if provided.
	if v, ok := d.GetOk("vm_id"); ok {
		uid, err := uuid.Parse(v.(string))
		if err != nil {
			return diag.Errorf("invalid vm_id: %s", err)
		}
		opts.VMID = &uid
	}
	items, _, err := client.Backups.List(ctx, opts)
	if err != nil {
		return diag.FromErr(err)
	}
	out := make([]map[string]any, 0, len(items))
	for i := range items {
		out = append(out, backupToMap(&items[i]))
	}
	d.SetId("backups")
	d.Set("backups", out)
	return nil
}

func backupToMap(b *raff.Backup) map[string]any {
	m := map[string]any{
		"id":           b.ID.String(),
		"name":         b.Name,
		"status":       b.Status,
		"storage_size": b.StorageSize,
	}
	if b.ProductVM != nil {
		m["vm_id"] = b.ProductVM.String()
	}
	if b.Region != nil {
		m["region"] = string(*b.Region)
	}
	if b.AccountID != nil {
		m["account_id"] = b.AccountID.String()
	}
	if b.ProjectID != nil {
		m["project_id"] = b.ProjectID.String()
	}
	if b.ExpireDate != nil {
		m["expire_date"] = b.ExpireDate.Format("2006-01-02T15:04:05Z")
	}
	if b.CreatedAt != nil {
		m["created_at"] = b.CreatedAt.Format("2006-01-02T15:04:05Z")
	}
	return m
}
