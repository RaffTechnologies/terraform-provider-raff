package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	raff "github.com/rafftechnologies/raff-go"
	"github.com/rafftechnologies/raff-go/spec"
)

// vmComputedSchema mirrors resourceVM but with everything Computed,
// for use in singular and plural data sources.
func vmComputedSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"id":                   {Type: schema.TypeString, Computed: true},
		"name":                 {Type: schema.TypeString, Computed: true},
		"status":               {Type: schema.TypeString, Computed: true},
		"region":               {Type: schema.TypeString, Computed: true},
		"template_id":          {Type: schema.TypeString, Computed: true},
		"template_name":        {Type: schema.TypeString, Computed: true},
		"template_version":     {Type: schema.TypeString, Computed: true},
		"pricing_id":           {Type: schema.TypeInt, Computed: true},
		"price_per_hour":       {Type: schema.TypeString, Computed: true},
		"cpu":                  {Type: schema.TypeInt, Computed: true},
		"ram":                  {Type: schema.TypeInt, Computed: true},
		"storage":              {Type: schema.TypeInt, Computed: true},
		"added_storage":        {Type: schema.TypeInt, Computed: true},
		"total_storage":        {Type: schema.TypeInt, Computed: true},
		"public_ipv4_address":  {Type: schema.TypeString, Computed: true},
		"private_ipv4_address": {Type: schema.TypeString, Computed: true},
		"public_ipv6_address":  {Type: schema.TypeString, Computed: true},
		"private_ipv6_address": {Type: schema.TypeString, Computed: true},
		"billing_type":         {Type: schema.TypeString, Computed: true},
		"active":               {Type: schema.TypeBool, Computed: true},
		"created_at":           {Type: schema.TypeString, Computed: true},
		"updated_at":           {Type: schema.TypeString, Computed: true},
	}
}

func dataSourceVM() *schema.Resource {
	s := vmComputedSchema()
	s["id"] = &schema.Schema{Type: schema.TypeString, Required: true}
	return &schema.Resource{
		Description: "Reads a single Raff VM by ID.",
		ReadContext: dataSourceVMRead,
		Schema:      s,
	}
}

func dataSourceVMRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	vm, _, err := client.VMs.Get(ctx, d.Get("id").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(vm.ID.String())
	flattenVMInto(d, vm)
	return nil
}

func dataSourceVMs() *schema.Resource {
	return &schema.Resource{
		Description: "Lists Raff VMs in the current project, with optional filters.",
		ReadContext: dataSourceVMsRead,
		Schema: map[string]*schema.Schema{
			"region": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"status": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"vms": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Resource{Schema: vmComputedSchema()},
			},
		},
	}
}

func dataSourceVMsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)

	opts := &spec.ListVMsParams{}
	if v, ok := d.GetOk("region"); ok {
		r := spec.ListVMsParamsRegion(v.(string))
		opts.Region = &r
	}
	if v, ok := d.GetOk("status"); ok {
		s := spec.ListVMsParamsStatus(v.(string))
		opts.Status = &s
	}

	vms, _, err := client.VMs.List(ctx, opts)
	if err != nil {
		return diag.FromErr(err)
	}

	out := make([]map[string]any, 0, len(vms))
	for i := range vms {
		out = append(out, vmToMap(&vms[i]))
	}
	d.SetId("vms")
	d.Set("vms", out)
	return nil
}

func vmToMap(vm *raff.VM) map[string]any {
	m := map[string]any{
		"id":               vm.ID.String(),
		"name":             vm.Name,
		"status":           string(vm.Status),
		"region":           string(vm.Region),
		"template_id":      vm.TemplateID.String(),
		"template_name":    vm.TemplateName,
		"template_version": vm.TemplateVersion,
		"pricing_id":       vm.PricingID,
		"price_per_hour":   vm.PricePerHour,
		"cpu":              vm.CPU,
		"ram":              vm.RAM,
		"storage":          vm.Storage,
		"added_storage":    vm.AddedStorage,
		"total_storage":    vm.TotalStorage,
		"active":           vm.Active,
		"created_at":       vm.CreatedAt.Format("2006-01-02T15:04:05Z"),
		"updated_at":       vm.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
	m["public_ipv4_address"] = raff.StringValue(vm.PublicIpv4Address)
	m["private_ipv4_address"] = raff.StringValue(vm.PrivateIpv4Address)
	m["public_ipv6_address"] = raff.StringValue(vm.PublicIpv6Address)
	m["private_ipv6_address"] = raff.StringValue(vm.PrivateIpv6Address)
	if vm.BillingType != nil {
		m["billing_type"] = string(*vm.BillingType)
	} else {
		m["billing_type"] = ""
	}
	return m
}

func flattenVMInto(d *schema.ResourceData, vm *raff.VM) {
	for k, v := range vmToMap(vm) {
		d.Set(k, v)
	}
}
