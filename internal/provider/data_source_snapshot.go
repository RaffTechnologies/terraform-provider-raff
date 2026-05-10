package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	raff "github.com/rafftechnologies/raff-go"
)

func snapshotComputedSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"id":            {Type: schema.TypeInt, Computed: true},
		"name":          {Type: schema.TypeString, Computed: true},
		"resource_type": {Type: schema.TypeString, Computed: true},
		"vm_id":         {Type: schema.TypeString, Computed: true},
		"size":          {Type: schema.TypeString, Computed: true},
		"status":        {Type: schema.TypeString, Computed: true},
		"created_at":    {Type: schema.TypeString, Computed: true},
	}
}

func dataSourceSnapshot() *schema.Resource {
	s := snapshotComputedSchema()
	s["id"] = &schema.Schema{Type: schema.TypeInt, Required: true}
	return &schema.Resource{
		Description: "Reads a single snapshot by ID.",
		ReadContext: dataSourceSnapshotRead,
		Schema:      s,
	}
}

func dataSourceSnapshotRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	s, _, err := client.Snapshots.Get(ctx, d.Get("id").(int))
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(itoa(s.ID))
	for k, v := range snapshotToMap(s) {
		d.Set(k, v)
	}
	return nil
}

func dataSourceSnapshots() *schema.Resource {
	return &schema.Resource{
		Description: "Lists snapshots in the current project.",
		ReadContext: dataSourceSnapshotsRead,
		Schema: map[string]*schema.Schema{
			"snapshots": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Resource{Schema: snapshotComputedSchema()},
			},
		},
	}
}

func dataSourceSnapshotsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	snaps, _, err := client.Snapshots.List(ctx, nil)
	if err != nil {
		return diag.FromErr(err)
	}
	out := make([]map[string]any, 0, len(snaps))
	for i := range snaps {
		out = append(out, snapshotToMap(&snaps[i]))
	}
	d.SetId("snapshots")
	d.Set("snapshots", out)
	return nil
}

func snapshotToMap(s *raff.Snapshot) map[string]any {
	m := map[string]any{
		"id":            s.ID,
		"name":          s.Name,
		"resource_type": string(s.Type),
	}
	if s.ProductVM != nil {
		m["vm_id"] = s.ProductVM.String()
	}
	if s.Size != nil {
		m["size"] = *s.Size
	}
	if s.Status != nil {
		m["status"] = string(*s.Status)
	}
	if s.CreatedAt != nil {
		m["created_at"] = s.CreatedAt.Format("2006-01-02T15:04:05Z")
	}
	return m
}
