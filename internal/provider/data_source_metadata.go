package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	raff "github.com/rafftechnologies/raff-go"
	"github.com/rafftechnologies/raff-go/spec"
)

// === regions ===

func dataSourceRegions() *schema.Resource {
	return &schema.Resource{
		Description: "Lists all available datacenter regions. Useful for picking a region argument when creating VMs / volumes / VPCs.",
		ReadContext: dataSourceRegionsRead,
		Schema: map[string]*schema.Schema{
			"regions": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{Schema: map[string]*schema.Schema{
					"code":         {Type: schema.TypeString, Computed: true},
					"name":         {Type: schema.TypeString, Computed: true},
					"country_code": {Type: schema.TypeString, Computed: true},
					"flag":         {Type: schema.TypeString, Computed: true},
					"is_default":   {Type: schema.TypeBool, Computed: true},
				}},
			},
		},
	}
}

func dataSourceRegionsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	regions, _, err := client.Metadata.ListRegions(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	out := make([]map[string]any, 0, len(regions))
	for _, r := range regions {
		m := map[string]any{
			"code":         r.Code,
			"name":         r.Name,
			"country_code": r.CountryCode,
			"is_default":   r.IsDefault,
		}
		if r.Flag != nil {
			m["flag"] = *r.Flag
		}
		out = append(out, m)
	}
	d.SetId("regions")
	d.Set("regions", out)
	return nil
}

// === templates ===

func dataSourceTemplates() *schema.Resource {
	return &schema.Resource{
		Description: "Lists OS templates available for VM creation. Mirrors the digitalocean_images data source pattern.",
		ReadContext: dataSourceTemplatesRead,
		Schema: map[string]*schema.Schema{
			"category": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Filter: linux, windows, app, etc.",
			},
			"vm_type": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Filter: standard or premium.",
			},
			"region": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Filter by region.",
			},
			"templates": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{Schema: map[string]*schema.Schema{
					"id":          {Type: schema.TypeString, Computed: true},
					"name":        {Type: schema.TypeString, Computed: true},
					"version":     {Type: schema.TypeString, Computed: true},
					"os_type":     {Type: schema.TypeString, Computed: true},
					"category":    {Type: schema.TypeString, Computed: true},
					"region":      {Type: schema.TypeString, Computed: true},
					"is_windows":  {Type: schema.TypeBool, Computed: true},
					"min_cpu":     {Type: schema.TypeInt, Computed: true},
					"min_ram":     {Type: schema.TypeInt, Computed: true},
					"min_storage": {Type: schema.TypeInt, Computed: true},
					"description": {Type: schema.TypeString, Computed: true},
				}},
			},
		},
	}
}

func dataSourceTemplatesRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	opts := &raff.TemplateListOptions{}
	if v, ok := d.GetOk("category"); ok {
		c := spec.ListTemplatesParamsCategory(v.(string))
		opts.Category = &c
	}
	if v, ok := d.GetOk("vm_type"); ok {
		t := spec.ListTemplatesParamsVMType(v.(string))
		opts.VMType = &t
	}
	if v, ok := d.GetOk("region"); ok {
		r := spec.ListTemplatesParamsRegion(v.(string))
		opts.Region = &r
	}
	templates, _, err := client.Metadata.ListTemplates(ctx, opts)
	if err != nil {
		return diag.FromErr(err)
	}
	out := make([]map[string]any, 0, len(templates))
	for _, t := range templates {
		m := map[string]any{
			"id":          t.ID.String(),
			"name":        t.Name,
			"version":     t.Version,
			"os_type":     t.OsType,
			"category":    string(t.Category),
			"region":      string(t.Region),
			"is_windows":  t.IsWindows,
			"min_cpu":     t.CPU,
			"min_ram":     t.RAM,
			"min_storage": t.Storage,
		}
		if t.Description != nil {
			m["description"] = *t.Description
		}
		out = append(out, m)
	}
	d.SetId("templates")
	d.Set("templates", out)
	return nil
}
