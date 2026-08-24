package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	raff "github.com/rafftechnologies/raff-go"
)

func appServiceComputedSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"id":                  {Type: schema.TypeString, Computed: true},
		"service_id":          {Type: schema.TypeString, Computed: true},
		"project_id":          {Type: schema.TypeString, Computed: true},
		"name":                {Type: schema.TypeString, Computed: true},
		"slug":                {Type: schema.TypeString, Computed: true},
		"description":         {Type: schema.TypeString, Computed: true},
		"service_type":        {Type: schema.TypeString, Computed: true},
		"source_type":         {Type: schema.TypeString, Computed: true},
		"builder":             {Type: schema.TypeString, Computed: true},
		"repo_full_name":      {Type: schema.TypeString, Computed: true},
		"repo_branch":         {Type: schema.TypeString, Computed: true},
		"image_ref":           {Type: schema.TypeString, Computed: true},
		"tier_id":             {Type: schema.TypeInt, Computed: true},
		"tier_name":           {Type: schema.TypeString, Computed: true},
		"replicas":            {Type: schema.TypeInt, Computed: true},
		"autoscaling_enabled": {Type: schema.TypeBool, Computed: true},
		"min_replicas":        {Type: schema.TypeInt, Computed: true},
		"max_replicas":        {Type: schema.TypeInt, Computed: true},
		"scale_to_zero":       {Type: schema.TypeBool, Computed: true},
		"region":              {Type: schema.TypeString, Computed: true},
		"url":                 {Type: schema.TypeString, Computed: true},
		"internal_host":       {Type: schema.TypeString, Computed: true},
		"status":              {Type: schema.TypeString, Computed: true},
		"status_message":      {Type: schema.TypeString, Computed: true},
		"billing_type":        {Type: schema.TypeString, Computed: true},
		"created_at":          {Type: schema.TypeString, Computed: true},
		"updated_at":          {Type: schema.TypeString, Computed: true},
	}
}

func appServiceToMap(s *raff.AppService) map[string]any {
	return map[string]any{
		"id":                  s.ID,
		"service_id":          s.ServiceID,
		"project_id":          s.ProjectID,
		"name":                s.Name,
		"slug":                s.Slug,
		"description":         s.Description,
		"service_type":        s.ServiceType,
		"source_type":         s.SourceType,
		"builder":             s.Builder,
		"repo_full_name":      s.RepoFullName,
		"repo_branch":         s.RepoBranch,
		"image_ref":           s.ImageRef,
		"tier_id":             s.TierID,
		"tier_name":           s.TierName,
		"replicas":            s.Replicas,
		"autoscaling_enabled": s.AutoscalingEnabled,
		"min_replicas":        s.MinReplicas,
		"max_replicas":        s.MaxReplicas,
		"scale_to_zero":       s.ScaleToZero,
		"region":              s.Region,
		"url":                 s.URL,
		"internal_host":       s.InternalHost,
		"status":              s.Status,
		"status_message":      s.StatusMessage,
		"billing_type":        s.BillingType,
		"created_at":          s.CreatedAt,
		"updated_at":          s.UpdatedAt,
	}
}

func dataSourceAppService() *schema.Resource {
	s := appServiceComputedSchema()
	s["id"] = &schema.Schema{Type: schema.TypeString, Required: true, Description: "App service ID or short id."}
	return &schema.Resource{
		Description: "Reads a single Raff Apps service by ID.",
		ReadContext: dataSourceAppServiceRead,
		Schema:      s,
	}
}

func dataSourceAppServiceRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	s, _, err := client.AppServices.Get(ctx, d.Get("id").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(s.ID)
	for k, v := range appServiceToMap(s) {
		d.Set(k, v)
	}
	return nil
}

func dataSourceAppServices() *schema.Resource {
	return &schema.Resource{
		Description: "Lists Raff Apps services in the current account.",
		ReadContext: dataSourceAppServicesRead,
		Schema: map[string]*schema.Schema{
			"services": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Resource{Schema: appServiceComputedSchema()},
			},
		},
	}
}

func dataSourceAppServicesRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	items, _, err := client.AppServices.List(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	out := make([]map[string]any, 0, len(items))
	for i := range items {
		out = append(out, appServiceToMap(&items[i]))
	}
	d.SetId("app_services")
	d.Set("services", out)
	return nil
}

func dataSourceAppTiers() *schema.Resource {
	return &schema.Resource{
		Description: "Lists Raff Apps pricing tiers (tier IDs to use as tier_id when creating raff_app_service).",
		ReadContext: dataSourceAppTiersRead,
		Schema: map[string]*schema.Schema{
			"tiers": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{Schema: map[string]*schema.Schema{
					"id":              {Type: schema.TypeInt, Computed: true},
					"name":            {Type: schema.TypeString, Computed: true},
					"vcpu":            {Type: schema.TypeFloat, Computed: true},
					"memory_mib":      {Type: schema.TypeInt, Computed: true},
					"ephemeral_gib":   {Type: schema.TypeInt, Computed: true},
					"price_per_hour":  {Type: schema.TypeFloat, Computed: true},
					"price_per_month": {Type: schema.TypeFloat, Computed: true},
					"yearly_price":    {Type: schema.TypeFloat, Computed: true},
					"region":          {Type: schema.TypeString, Computed: true},
				}},
			},
		},
	}
}

func dataSourceAppTiersRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	tiers, _, err := client.AppServices.ListTiers(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	out := make([]map[string]any, 0, len(tiers))
	for _, t := range tiers {
		out = append(out, map[string]any{
			"id":              t.ID,
			"name":            t.Name,
			"vcpu":            t.VCPU,
			"memory_mib":      t.MemoryMiB,
			"ephemeral_gib":   t.EphemeralGiB,
			"price_per_hour":  t.PricePerHour,
			"price_per_month": t.PricePerMonth,
			"yearly_price":    t.YearlyPrice,
			"region":          t.Region,
		})
	}
	d.SetId("app_tiers")
	d.Set("tiers", out)
	return nil
}
