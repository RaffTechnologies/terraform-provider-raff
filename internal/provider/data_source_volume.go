package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	raff "github.com/rafftechnologies/raff-go"
)

func volumeComputedSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"id":             {Type: schema.TypeInt, Computed: true},
		"name":           {Type: schema.TypeString, Computed: true},
		"size":           {Type: schema.TypeInt, Computed: true},
		"volume_type":    {Type: schema.TypeString, Computed: true},
		"region":         {Type: schema.TypeString, Computed: true},
		"status":         {Type: schema.TypeString, Computed: true},
		"vm_id":          {Type: schema.TypeString, Computed: true},
		"price_per_hour": {Type: schema.TypeString, Computed: true},
		"account_id":     {Type: schema.TypeString, Computed: true},
		"project_id":     {Type: schema.TypeString, Computed: true},
		"created_at":     {Type: schema.TypeString, Computed: true},
		"updated_at":     {Type: schema.TypeString, Computed: true},
	}
}

func dataSourceVolume() *schema.Resource {
	s := volumeComputedSchema()
	s["id"] = &schema.Schema{Type: schema.TypeInt, Required: true}
	return &schema.Resource{
		Description: "Reads a single volume by ID.",
		ReadContext: dataSourceVolumeRead,
		Schema:      s,
	}
}

func dataSourceVolumeRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	v, _, err := client.Volumes.Get(ctx, d.Get("id").(int))
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(itoa(v.ID))
	for k, vv := range volumeToMap(v) {
		d.Set(k, vv)
	}
	return nil
}

func dataSourceVolumes() *schema.Resource {
	return &schema.Resource{
		Description: "Lists volumes in the current project.",
		ReadContext: dataSourceVolumesRead,
		Schema: map[string]*schema.Schema{
			"volumes": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Resource{Schema: volumeComputedSchema()},
			},
		},
	}
}

func dataSourceVolumesRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	vols, _, err := client.Volumes.List(ctx, nil)
	if err != nil {
		return diag.FromErr(err)
	}
	out := make([]map[string]any, 0, len(vols))
	for i := range vols {
		out = append(out, volumeToMap(&vols[i]))
	}
	d.SetId("volumes")
	d.Set("volumes", out)
	return nil
}

func volumeToMap(v *raff.Volume) map[string]any {
	m := map[string]any{
		"id":             v.ID,
		"name":           v.Name,
		"size":           v.Size,
		"volume_type":    string(v.VolumeType),
		"status":         string(v.Status),
		"price_per_hour": raff.StringValue(v.PricePerHour),
	}
	if v.Region != nil {
		m["region"] = string(*v.Region)
	}
	if v.ProductVM != nil {
		m["vm_id"] = v.ProductVM.String()
	}
	if v.AccountID != nil {
		m["account_id"] = v.AccountID.String()
	}
	if v.ProjectID != nil {
		m["project_id"] = v.ProjectID.String()
	}
	if v.CreatedAt != nil {
		m["created_at"] = v.CreatedAt.Format("2006-01-02T15:04:05Z")
	}
	if v.UpdatedAt != nil {
		m["updated_at"] = v.UpdatedAt.Format("2006-01-02T15:04:05Z")
	}
	return m
}
