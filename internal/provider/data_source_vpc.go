package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	raff "github.com/rafftechnologies/raff-go"
)

func vpcComputedSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"id":               {Type: schema.TypeString, Computed: true},
		"name":             {Type: schema.TypeString, Computed: true},
		"cidr":             {Type: schema.TypeString, Computed: true},
		"region":           {Type: schema.TypeString, Computed: true},
		"status":           {Type: schema.TypeString, Computed: true},
		"dns":              {Type: schema.TypeString, Computed: true},
		"gateway":          {Type: schema.TypeString, Computed: true},
		"gateway_type":     {Type: schema.TypeString, Computed: true},
		"router_public_ip": {Type: schema.TypeString, Computed: true},
		"router_status":    {Type: schema.TypeString, Computed: true},
		"account_id":       {Type: schema.TypeString, Computed: true},
		"project_id":       {Type: schema.TypeString, Computed: true},
		"total_ips":        {Type: schema.TypeInt, Computed: true},
		"used_ips":         {Type: schema.TypeInt, Computed: true},
		"created_at":       {Type: schema.TypeString, Computed: true},
		"updated_at":       {Type: schema.TypeString, Computed: true},
	}
}

func dataSourceVPC() *schema.Resource {
	s := vpcComputedSchema()
	s["id"] = &schema.Schema{Type: schema.TypeString, Required: true}
	return &schema.Resource{
		Description: "Reads a single Raff VPC by ID.",
		ReadContext: dataSourceVPCRead,
		Schema:      s,
	}
}

func dataSourceVPCRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	vpc, _, err := client.VPCs.Get(ctx, d.Get("id").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(vpc.ID.String())
	for k, v := range vpcToMap(vpc) {
		d.Set(k, v)
	}
	return nil
}

func dataSourceVPCs() *schema.Resource {
	return &schema.Resource{
		Description: "Lists Raff VPCs in the current project.",
		ReadContext: dataSourceVPCsRead,
		Schema: map[string]*schema.Schema{
			"vpcs": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Resource{Schema: vpcComputedSchema()},
			},
		},
	}
}

func dataSourceVPCsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	vpcs, _, err := client.VPCs.List(ctx, nil)
	if err != nil {
		return diag.FromErr(err)
	}
	out := make([]map[string]any, 0, len(vpcs))
	for i := range vpcs {
		out = append(out, vpcToMap(&vpcs[i]))
	}
	d.SetId("vpcs")
	d.Set("vpcs", out)
	return nil
}

func vpcToMap(v *raff.VPC) map[string]any {
	m := map[string]any{
		"id":               v.ID.String(),
		"name":             v.Name,
		"cidr":             v.Cidr,
		"region":           string(v.Region),
		"status":           v.Status,
		"dns":              raff.StringValue(v.DNS),
		"gateway":          raff.StringValue(v.Gateway),
		"gateway_type":     raff.StringValue(v.GatewayType),
		"router_public_ip": raff.StringValue(v.RouterPublicIP),
		"router_status":    raff.StringValue(v.RouterStatus),
	}
	if v.AccountID != nil {
		m["account_id"] = v.AccountID.String()
	}
	if v.ProjectID != nil {
		m["project_id"] = v.ProjectID.String()
	}
	if v.TotalIps != nil {
		m["total_ips"] = *v.TotalIps
	}
	if v.UsedIps != nil {
		m["used_ips"] = *v.UsedIps
	}
	if v.CreatedAt != nil {
		m["created_at"] = v.CreatedAt.Format("2006-01-02T15:04:05Z")
	}
	if v.UpdatedAt != nil {
		m["updated_at"] = v.UpdatedAt.Format("2006-01-02T15:04:05Z")
	}
	return m
}
