package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	raff "github.com/rafftechnologies/raff-go"
	"github.com/rafftechnologies/raff-go/spec"
)

// dataSourceVMPricing mirrors the digitalocean_sizes pattern: a list of
// pricing plans for sizing decisions when creating VMs.
func dataSourceVMPricing() *schema.Resource {
	return &schema.Resource{
		Description: "Lists VM pricing plans (size IDs to use as pricing_id when creating raff_vm).",
		ReadContext: dataSourceVMPricingRead,
		Schema: map[string]*schema.Schema{
			"region":  {Type: schema.TypeString, Optional: true},
			"vm_type": {Type: schema.TypeString, Optional: true, Description: "Filter: standard or premium."},
			"plans": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{Schema: map[string]*schema.Schema{
					"id":                      {Type: schema.TypeInt, Computed: true},
					"vm_type":                 {Type: schema.TypeString, Computed: true},
					"region":                  {Type: schema.TypeString, Computed: true},
					"vcpu":                    {Type: schema.TypeInt, Computed: true},
					"memory_gib":              {Type: schema.TypeInt, Computed: true},
					"ssd_gib":                 {Type: schema.TypeInt, Computed: true},
					"transfer_gib":            {Type: schema.TypeInt, Computed: true},
					"price_per_hour":          {Type: schema.TypeFloat, Computed: true},
					"monthly_price":           {Type: schema.TypeFloat, Computed: true},
					"yearly_price":            {Type: schema.TypeFloat, Computed: true},
					"twenty_four_month_price": {Type: schema.TypeFloat, Computed: true},
				}},
			},
		},
	}
}

func dataSourceVMPricingRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	opts := &raff.VMPricingListOptions{}
	if v, ok := d.GetOk("region"); ok {
		r := spec.ListVMPricingParamsRegion(v.(string))
		opts.Region = &r
	}
	if v, ok := d.GetOk("vm_type"); ok {
		t := spec.ListVMPricingParamsType(v.(string))
		opts.Type = &t
	}
	plans, _, err := client.Pricing.ListVM(ctx, opts)
	if err != nil {
		return diag.FromErr(err)
	}
	out := make([]map[string]any, 0, len(plans))
	for _, p := range plans {
		out = append(out, map[string]any{
			"id":                      p.ID,
			"vm_type":                 string(p.VMType),
			"region":                  string(p.Region),
			"vcpu":                    p.Vcpu,
			"memory_gib":              p.MemoryGib,
			"ssd_gib":                 p.SsdGib,
			"transfer_gib":            p.TransferGib,
			"price_per_hour":          p.PricePerHour,
			"monthly_price":           p.MonthlyPrice,
			"yearly_price":            p.YearlyPrice,
			"twenty_four_month_price": p.TwentyFourMonthPrice,
		})
	}
	d.SetId("vm_pricing")
	d.Set("plans", out)
	return nil
}

// dataSourceStoragePricing handles volume / backup / snapshot pricing —
// they all share the same StoragePricing response shape (per-GB rates).
func dataSourceStoragePricing(kind string) *schema.Resource {
	return &schema.Resource{
		Description: "Per-GB " + kind + " storage pricing for the requested region (or default region if omitted).",
		ReadContext: storagePricingReader(kind),
		Schema: map[string]*schema.Schema{
			"region":              {Type: schema.TypeString, Optional: true},
			"price_per_gb_hour":   {Type: schema.TypeFloat, Computed: true},
			"price_per_gb_month":  {Type: schema.TypeFloat, Computed: true},
			"effective_region":    {Type: schema.TypeString, Computed: true},
		},
	}
}

func storagePricingReader(kind string) func(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	return func(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
		client := meta.(*raff.Client)
		region, _ := d.Get("region").(string)
		var (
			p   *raff.StoragePricing
			err error
		)
		switch kind {
		case "volume":
			opts := &raff.VolumePricingListOptions{}
			if region != "" {
				r := spec.ListVolumePricingParamsRegion(region)
				opts.Region = &r
			}
			p, _, err = client.Pricing.ListVolume(ctx, opts)
		case "backup":
			opts := &raff.BackupPricingListOptions{}
			if region != "" {
				r := spec.ListBackupPricingParamsRegion(region)
				opts.Region = &r
			}
			p, _, err = client.Pricing.ListBackup(ctx, opts)
		case "snapshot":
			opts := &raff.SnapshotPricingListOptions{}
			if region != "" {
				r := spec.ListSnapshotPricingParamsRegion(region)
				opts.Region = &r
			}
			p, _, err = client.Pricing.ListSnapshot(ctx, opts)
		}
		if err != nil {
			return diag.FromErr(err)
		}
		d.SetId(kind + "_pricing")
		if p.PricePerGbHour != nil {
			d.Set("price_per_gb_hour", *p.PricePerGbHour)
		}
		if p.PricePerGbMonth != nil {
			d.Set("price_per_gb_month", *p.PricePerGbMonth)
		}
		if p.Region != nil {
			d.Set("effective_region", string(*p.Region))
		}
		return nil
	}
}

// dataSourceIPPricing is structured differently — IPPricing groups by
// IP family rather than per-GB.
func dataSourceIPPricing() *schema.Resource {
	tier := func() *schema.Resource {
		return &schema.Resource{Schema: map[string]*schema.Schema{
			"price_per_hour":          {Type: schema.TypeFloat, Computed: true},
			"monthly_price":           {Type: schema.TypeFloat, Computed: true},
			"yearly_price":            {Type: schema.TypeFloat, Computed: true},
			"twenty_four_month_price": {Type: schema.TypeFloat, Computed: true},
		}}
	}
	return &schema.Resource{
		Description: "Floating IP pricing by family (ipv4 / ipv6).",
		ReadContext: dataSourceIPPricingRead,
		Schema: map[string]*schema.Schema{
			"ipv4": {Type: schema.TypeList, Computed: true, Elem: tier()},
			"ipv6": {Type: schema.TypeList, Computed: true, Elem: tier()},
		},
	}
}

func dataSourceIPPricingRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	p, _, err := client.Pricing.ListIP(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId("ip_pricing")
	if p.Ipv4 != nil {
		d.Set("ipv4", []map[string]any{ipPricingTierMap(p.Ipv4)})
	}
	if p.Ipv6 != nil {
		d.Set("ipv6", []map[string]any{ipPricingTierMap(p.Ipv6)})
	}
	return nil
}

func ipPricingTierMap(t *spec.IPPricingTier) map[string]any {
	m := map[string]any{}
	if t.PricePerHour != nil {
		m["price_per_hour"] = *t.PricePerHour
	}
	if t.MonthlyPrice != nil {
		m["monthly_price"] = *t.MonthlyPrice
	}
	if t.YearlyPrice != nil {
		m["yearly_price"] = *t.YearlyPrice
	}
	if t.TwentyFourMonthPrice != nil {
		m["twenty_four_month_price"] = *t.TwentyFourMonthPrice
	}
	return m
}

