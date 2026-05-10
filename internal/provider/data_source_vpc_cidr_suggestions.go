package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	raff "github.com/rafftechnologies/raff-go"
)

// dataSourceVPCCIDRSuggestions exposes /vpcs/cidr-suggestions so customers
// can declaratively pick a non-overlapping CIDR for a new VPC instead of
// hardcoding one and risking conflicts with their existing VPCs.
func dataSourceVPCCIDRSuggestions() *schema.Resource {
	return &schema.Resource{
		Description: "Returns a recommended CIDR (and alternatives) that does not overlap your existing VPCs. Useful for declaratively sizing a new VPC without hardcoding a block that might conflict.",
		ReadContext: dataSourceVPCCIDRSuggestionsRead,
		Schema: map[string]*schema.Schema{
			"suggested": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Recommended CIDR — non-overlapping, right-sized for typical use.",
			},
			"suggested_ips": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Number of usable IPs in the suggested block.",
			},
			"existing_count": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Number of VPCs the account already has.",
			},
			"alternatives": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Non-overlapping CIDRs sized small / medium / large.",
				Elem: &schema.Resource{Schema: map[string]*schema.Schema{
					"cidr":          {Type: schema.TypeString, Computed: true},
					"size":          {Type: schema.TypeString, Computed: true},
					"available_ips": {Type: schema.TypeInt, Computed: true},
				}},
			},
		},
	}
}

func dataSourceVPCCIDRSuggestionsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	resp, _, err := client.VPCs.CIDRSuggestions(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("vpc_cidr_suggestions")
	d.Set("suggested", raff.StringValue(resp.Suggested))
	if resp.SuggestedIps != nil {
		d.Set("suggested_ips", *resp.SuggestedIps)
	}
	if resp.ExistingCount != nil {
		d.Set("existing_count", *resp.ExistingCount)
	}

	alts := make([]map[string]any, 0)
	if resp.Alternatives != nil {
		for _, a := range *resp.Alternatives {
			size := ""
			if a.Size != nil {
				size = string(*a.Size)
			}
			row := map[string]any{
				"cidr": raff.StringValue(a.Cidr),
				"size": size,
			}
			if a.AvailableIps != nil {
				row["available_ips"] = *a.AvailableIps
			}
			alts = append(alts, row)
		}
	}
	d.Set("alternatives", alts)
	return nil
}
