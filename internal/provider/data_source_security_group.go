package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	raff "github.com/rafftechnologies/raff-go"
)

func securityGroupComputedSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"id":          {Type: schema.TypeString, Computed: true},
		"name":        {Type: schema.TypeString, Computed: true},
		"description": {Type: schema.TypeString, Computed: true},
		"project_id":  {Type: schema.TypeString, Computed: true},
		"vm_count":    {Type: schema.TypeInt, Computed: true},
		"created_at":  {Type: schema.TypeString, Computed: true},
		"updated_at":  {Type: schema.TypeString, Computed: true},
		"rule": {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"rule_type": {Type: schema.TypeString, Computed: true},
					"protocol":  {Type: schema.TypeString, Computed: true},
					"range":     {Type: schema.TypeString, Computed: true},
					"ip":        {Type: schema.TypeString, Computed: true},
					"size":      {Type: schema.TypeInt, Computed: true},
					"icmp_type": {Type: schema.TypeInt, Computed: true},
				},
			},
		},
	}
}

func dataSourceSecurityGroup() *schema.Resource {
	s := securityGroupComputedSchema()
	s["id"] = &schema.Schema{Type: schema.TypeString, Required: true}
	return &schema.Resource{
		Description: "Reads a single Raff security group by ID.",
		ReadContext: dataSourceSecurityGroupRead,
		Schema:      s,
	}
}

func dataSourceSecurityGroupRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	sg, _, err := client.SecurityGroups.Get(ctx, d.Get("id").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(sg.ID.String())
	for k, v := range securityGroupToMap(sg) {
		d.Set(k, v)
	}
	return nil
}

func dataSourceSecurityGroups() *schema.Resource {
	return &schema.Resource{
		Description: "Lists Raff security groups in the current project.",
		ReadContext: dataSourceSecurityGroupsRead,
		Schema: map[string]*schema.Schema{
			"security_groups": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Resource{Schema: securityGroupComputedSchema()},
			},
		},
	}
}

func dataSourceSecurityGroupsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	sgs, _, err := client.SecurityGroups.List(ctx, nil)
	if err != nil {
		return diag.FromErr(err)
	}
	out := make([]map[string]any, 0, len(sgs))
	for i := range sgs {
		out = append(out, securityGroupToMap(&sgs[i]))
	}
	d.SetId("security_groups")
	d.Set("security_groups", out)
	return nil
}

func securityGroupToMap(sg *raff.SecurityGroup) map[string]any {
	m := map[string]any{
		"id":          sg.ID.String(),
		"name":        sg.Name,
		"description": raff.StringValue(sg.Description),
		"rule":        flattenSecurityGroupRules(sg.Rules),
	}
	if sg.ProjectID != nil {
		m["project_id"] = sg.ProjectID.String()
	}
	if sg.VMCount != nil {
		m["vm_count"] = *sg.VMCount
	}
	if sg.CreatedAt != nil {
		m["created_at"] = sg.CreatedAt.Format("2006-01-02T15:04:05Z")
	}
	if sg.UpdatedAt != nil {
		m["updated_at"] = sg.UpdatedAt.Format("2006-01-02T15:04:05Z")
	}
	return m
}
