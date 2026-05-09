package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	raff "github.com/rafftechnologies/raff-go"
	"github.com/rafftechnologies/raff-go/spec"
)

func ipComputedSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"id":         {Type: schema.TypeString, Computed: true},
		"ip_address": {Type: schema.TypeString, Computed: true},
		"type":       {Type: schema.TypeString, Computed: true},
		"region":     {Type: schema.TypeString, Computed: true},
		"status":     {Type: schema.TypeString, Computed: true},
		"reserved":   {Type: schema.TypeBool, Computed: true},
		"account_id": {Type: schema.TypeString, Computed: true},
		"project_id": {Type: schema.TypeString, Computed: true},
		"created_at": {Type: schema.TypeString, Computed: true},
		"updated_at": {Type: schema.TypeString, Computed: true},
	}
}

func dataSourceIP() *schema.Resource {
	s := ipComputedSchema()
	s["id"] = &schema.Schema{Type: schema.TypeString, Required: true}
	return &schema.Resource{
		Description: "Reads a single Raff floating IP by ID.",
		ReadContext: dataSourceIPRead,
		Schema:      s,
	}
}

func dataSourceIPRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	ip, _, err := client.IPs.Get(ctx, d.Get("id").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(ip.ID.String())
	for k, v := range ipToMap(ip) {
		d.Set(k, v)
	}
	return nil
}

func dataSourceIPs() *schema.Resource {
	return &schema.Resource{
		Description: "Lists Raff floating IPs in the current project.",
		ReadContext: dataSourceIPsRead,
		Schema: map[string]*schema.Schema{
			"status": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Filter by status (free or in-use).",
			},
			"reserved": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "When true, return only reserved IPs.",
			},
			"ips": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Resource{Schema: ipComputedSchema()},
			},
		},
	}
}

func dataSourceIPsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)

	opts := &spec.ListIPsParams{}
	if v, ok := d.GetOk("status"); ok {
		st := spec.ListIPsParamsStatus(v.(string))
		opts.Status = &st
	}
	if v, ok := d.GetOkExists("reserved"); ok {
		b := v.(bool)
		opts.Reserved = &b
	}

	ips, _, err := client.IPs.List(ctx, opts)
	if err != nil {
		return diag.FromErr(err)
	}
	out := make([]map[string]any, 0, len(ips))
	for i := range ips {
		out = append(out, ipToMap(&ips[i]))
	}
	d.SetId("ips")
	d.Set("ips", out)
	return nil
}

func ipToMap(ip *raff.FloatingIP) map[string]any {
	m := map[string]any{
		"id":         ip.ID.String(),
		"ip_address": ip.IPAddress,
		"type":       string(ip.Type),
		"status":     string(ip.Status),
		"reserved":   ip.Reserved,
	}
	if ip.Region != nil {
		m["region"] = string(*ip.Region)
	}
	if ip.AccountID != nil {
		m["account_id"] = ip.AccountID.String()
	}
	if ip.ProjectID != nil {
		m["project_id"] = ip.ProjectID.String()
	}
	if ip.CreatedAt != nil {
		m["created_at"] = ip.CreatedAt.Format("2006-01-02T15:04:05Z")
	}
	if ip.UpdatedAt != nil {
		m["updated_at"] = ip.UpdatedAt.Format("2006-01-02T15:04:05Z")
	}
	return m
}
