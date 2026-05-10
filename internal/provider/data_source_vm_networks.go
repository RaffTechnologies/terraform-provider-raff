package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	raff "github.com/rafftechnologies/raff-go"
)

// dataSourceVMNetworks lists every network interface attached to a VM.
// Useful for wiring downstream resources to a specific NIC by MAC, or for
// pulling the public IPv4 / VPC IPv4 / IPv6 of a VM into outputs.
func dataSourceVMNetworks() *schema.Resource {
	return &schema.Resource{
		Description: "Lists the network interfaces attached to a Raff VM. Returns one entry per NIC with stable MAC, IP, gateway, and security-group binding.",
		ReadContext: dataSourceVMNetworksRead,
		Schema: map[string]*schema.Schema{
			"vm_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "VM UUID to look up network interfaces for.",
			},
			"type": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Filter by interface type: `public`, `vpc`, or `ipv6`.",
			},
			"networks": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{Schema: map[string]*schema.Schema{
					"nic_id":            {Type: schema.TypeInt, Computed: true},
					"type":              {Type: schema.TypeString, Computed: true},
					"network_name":      {Type: schema.TypeString, Computed: true},
					"ip":                {Type: schema.TypeString, Computed: true},
					"mac":               {Type: schema.TypeString, Computed: true},
					"gateway":           {Type: schema.TypeString, Computed: true},
					"security_group_id": {Type: schema.TypeString, Computed: true},
				}},
			},
		},
	}
}

func dataSourceVMNetworksRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	vmID := d.Get("vm_id").(string)

	opts := &raff.ListVMNetworksOptions{}
	if v, ok := d.GetOk("type"); ok {
		t := raff.VMNetworkType(v.(string))
		opts.Type = &t
	}

	nets, _, err := client.VMs.ListNetworks(ctx, vmID, opts)
	if err != nil {
		return diag.FromErr(err)
	}

	out := make([]map[string]any, 0, len(nets))
	for _, n := range nets {
		row := map[string]any{
			"nic_id":       n.NicID,
			"type":         string(n.Type),
			"network_name": n.NetworkName,
			"ip":           n.IP,
			"mac":          raff.StringValue(n.Mac),
			"gateway":      raff.StringValue(n.Gateway),
		}
		if n.SecurityGroupID != nil {
			row["security_group_id"] = n.SecurityGroupID.String()
		}
		out = append(out, row)
	}

	d.SetId(vmID)
	d.Set("networks", out)
	return nil
}
