package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	raff "github.com/rafftechnologies/raff-go"
)

// dataSourceSecurityGroupTemplates lists the built-in security-group templates.
// `raff_security_group` already accepts `template_id`; this data source lets
// customers look up template IDs declaratively instead of hardcoding them.
func dataSourceSecurityGroupTemplates() *schema.Resource {
	return &schema.Resource{
		Description: "Lists the built-in security-group templates. Use a template's `id` as `template_id` on `raff_security_group` to clone its rules at creation time.",
		ReadContext: dataSourceSecurityGroupTemplatesRead,
		Schema: map[string]*schema.Schema{
			"templates": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{Schema: map[string]*schema.Schema{
					"id":          {Type: schema.TypeString, Computed: true},
					"name":        {Type: schema.TypeString, Computed: true},
					"description": {Type: schema.TypeString, Computed: true},
					"rule": {
						Type:     schema.TypeList,
						Computed: true,
						Elem: &schema.Resource{Schema: map[string]*schema.Schema{
							"rule_type": {Type: schema.TypeString, Computed: true},
							"protocol":  {Type: schema.TypeString, Computed: true},
							"range":     {Type: schema.TypeString, Computed: true},
							"ip":        {Type: schema.TypeString, Computed: true},
							"icmp_type": {Type: schema.TypeInt, Computed: true},
						}},
					},
				}},
			},
		},
	}
}

func dataSourceSecurityGroupTemplatesRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	tmpls, _, err := client.SecurityGroups.Templates(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	out := make([]map[string]any, 0, len(tmpls))
	for _, t := range tmpls {
		rules := make([]map[string]any, 0, len(t.Rules))
		for _, r := range t.Rules {
			rule := map[string]any{
				"rule_type": string(r.RuleType),
				"protocol":  string(r.Protocol),
				"range":     raff.StringValue(r.Range),
				"ip":        raff.StringValue(r.IP),
			}
			if r.IcmpType != nil {
				rule["icmp_type"] = *r.IcmpType
			}
			rules = append(rules, rule)
		}
		out = append(out, map[string]any{
			"id":          t.ID,
			"name":        t.Name,
			"description": raff.StringValue(t.Description),
			"rule":        rules,
		})
	}

	d.SetId("security_group_templates")
	d.Set("templates", out)
	return nil
}
