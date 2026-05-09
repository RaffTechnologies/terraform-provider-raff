package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	raff "github.com/rafftechnologies/raff-go"
	"github.com/rafftechnologies/raff-go/spec"
)

func resourceSecurityGroup() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a Raff security group — a named set of inbound/outbound rules attached to VM NICs.",
		CreateContext: resourceSecurityGroupCreate,
		ReadContext:   resourceSecurityGroupRead,
		UpdateContext: resourceSecurityGroupUpdate,
		DeleteContext: resourceSecurityGroupDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Security group name.",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Security group description.",
			},
			"template_id": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Seed from a template at create time. Template rules are copied, then merged with any explicit `rule` blocks.",
			},
			"rule": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Inbound and outbound rules. Updates replace the entire rule set.",
				Elem:        securityGroupRuleSchema(),
			},
			// Computed
			"project_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"vm_count": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Number of VM NICs currently using this security group.",
			},
			"created_at": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"updated_at": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func securityGroupRuleSchema() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"rule_type": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Direction of traffic (inbound or outbound).",
			},
			"protocol": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Network protocol (TCP, UDP, ICMP, ICMPV6, ALL).",
			},
			"range": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Port or port range. Single port (`80`) or range (`8000:9000`). Empty for ICMP/ALL.",
			},
			"ip": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Source/destination IP. Empty means any.",
			},
			"size": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "CIDR block size (e.g. `24` for /24). Used with `ip`.",
			},
			"icmp_type": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "ICMP message type (only for ICMP/ICMPV6).",
			},
		},
	}
}

func resourceSecurityGroupCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)

	req := &raff.CreateSecurityGroupRequest{Name: d.Get("name").(string)}
	if v := d.Get("description").(string); v != "" {
		req.Description = raff.String(v)
	}
	if v, ok := d.GetOk("template_id"); ok {
		req.TemplateID = raff.String(v.(string))
	}
	if rules := expandSecurityGroupRules(d.Get("rule").([]any)); rules != nil {
		req.Rules = rules
	}

	sg, _, err := client.SecurityGroups.Create(ctx, req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(sg.ID.String())

	return setSecurityGroupState(d, sg)
}

func resourceSecurityGroupRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)

	sg, _, err := client.SecurityGroups.Get(ctx, d.Id())
	if err != nil {
		if errResp, ok := err.(*raff.ErrorResponse); ok && errResp.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	return setSecurityGroupState(d, sg)
}

func resourceSecurityGroupUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)

	req := &raff.UpdateSecurityGroupRequest{}
	if d.HasChange("name") {
		req.Name = raff.String(d.Get("name").(string))
	}
	if d.HasChange("description") {
		req.Description = raff.String(d.Get("description").(string))
	}
	if d.HasChange("rule") {
		rules := expandSecurityGroupRules(d.Get("rule").([]any))
		if rules == nil {
			empty := []spec.SecurityGroupRule{}
			rules = &empty
		}
		req.Rules = rules
	}

	sg, _, err := client.SecurityGroups.Update(ctx, d.Id(), req)
	if err != nil {
		return diag.FromErr(err)
	}

	return setSecurityGroupState(d, sg)
}

func resourceSecurityGroupDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)

	_, err := client.SecurityGroups.Delete(ctx, d.Id())
	if err != nil {
		if errResp, ok := err.(*raff.ErrorResponse); ok && errResp.StatusCode == 404 {
			return nil
		}
		return diag.FromErr(err)
	}

	return nil
}

func expandSecurityGroupRules(in []any) *[]spec.SecurityGroupRule {
	if len(in) == 0 {
		return nil
	}
	out := make([]spec.SecurityGroupRule, 0, len(in))
	for _, raw := range in {
		m := raw.(map[string]any)
		rule := spec.SecurityGroupRule{
			RuleType: spec.SecurityGroupRuleRuleType(m["rule_type"].(string)),
			Protocol: spec.SecurityGroupRuleProtocol(m["protocol"].(string)),
		}
		if v, ok := m["range"].(string); ok && v != "" {
			rule.Range = raff.String(v)
		}
		if v, ok := m["ip"].(string); ok && v != "" {
			rule.IP = raff.String(v)
		}
		if v, ok := m["size"].(int); ok && v > 0 {
			rule.Size = raff.Int(v)
		}
		if v, ok := m["icmp_type"].(int); ok && v > 0 {
			rule.IcmpType = raff.Int(v)
		}
		out = append(out, rule)
	}
	return &out
}

func flattenSecurityGroupRules(rules []spec.SecurityGroupRule) []map[string]any {
	out := make([]map[string]any, 0, len(rules))
	for _, r := range rules {
		m := map[string]any{
			"rule_type": string(r.RuleType),
			"protocol":  string(r.Protocol),
		}
		if r.Range != nil {
			m["range"] = *r.Range
		}
		if r.IP != nil {
			m["ip"] = *r.IP
		}
		if r.Size != nil {
			m["size"] = *r.Size
		}
		if r.IcmpType != nil {
			m["icmp_type"] = *r.IcmpType
		}
		out = append(out, m)
	}
	return out
}

func setSecurityGroupState(d *schema.ResourceData, sg *raff.SecurityGroup) diag.Diagnostics {
	d.Set("name", sg.Name)
	d.Set("description", raff.StringValue(sg.Description))
	d.Set("rule", flattenSecurityGroupRules(sg.Rules))
	if sg.ProjectID != nil {
		d.Set("project_id", sg.ProjectID.String())
	}
	if sg.VMCount != nil {
		d.Set("vm_count", *sg.VMCount)
	}
	if sg.CreatedAt != nil {
		d.Set("created_at", sg.CreatedAt.Format("2006-01-02T15:04:05Z"))
	}
	if sg.UpdatedAt != nil {
		d.Set("updated_at", sg.UpdatedAt.Format("2006-01-02T15:04:05Z"))
	}

	return nil
}
