package provider

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	raff "github.com/rafftechnologies/raff-go"
)

func apiKeyComputedSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"id":              {Type: schema.TypeString, Computed: true},
		"name":            {Type: schema.TypeString, Computed: true},
		"key_prefix":      {Type: schema.TypeString, Computed: true},
		"is_active":       {Type: schema.TypeBool, Computed: true},
		"rate_limit_tier": {Type: schema.TypeString, Computed: true},
		"expires_at":      {Type: schema.TypeString, Computed: true},
		"created_at":      {Type: schema.TypeString, Computed: true},
	}
}

func dataSourceAPIKey() *schema.Resource {
	s := apiKeyComputedSchema()
	s["id"] = &schema.Schema{Type: schema.TypeString, Required: true}
	return &schema.Resource{
		Description: "Reads a single API key by ID. The plaintext secret is never returned by Get — only by Create or Regenerate.",
		ReadContext: dataSourceAPIKeyRead,
		Schema:      s,
	}
}

func dataSourceAPIKeyRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	k, _, err := client.APIKeys.Get(ctx, d.Get("id").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(k.ID.String())
	for kk, vv := range apiKeyToMap(k) {
		d.Set(kk, vv)
	}
	return nil
}

func dataSourceAPIKeys() *schema.Resource {
	return &schema.Resource{
		Description: "Lists API keys for the account (without secrets).",
		ReadContext: dataSourceAPIKeysRead,
		Schema: map[string]*schema.Schema{
			"api_keys": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Resource{Schema: apiKeyComputedSchema()},
			},
		},
	}
}

func dataSourceAPIKeysRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	keys, _, err := client.APIKeys.List(ctx, nil)
	if err != nil {
		return diag.FromErr(err)
	}
	out := make([]map[string]any, 0, len(keys))
	for i := range keys {
		out = append(out, apiKeyToMap(&keys[i]))
	}
	d.SetId("api_keys")
	d.Set("api_keys", out)
	return nil
}

func apiKeyToMap(k *raff.APIKey) map[string]any {
	m := map[string]any{
		"id":         k.ID.String(),
		"name":       k.Name,
		"key_prefix": k.KeyPrefix,
		"is_active":  k.IsActive,
	}
	if k.RateLimitTier != nil {
		m["rate_limit_tier"] = string(*k.RateLimitTier)
	}
	if k.ExpiresAt != nil {
		m["expires_at"] = k.ExpiresAt.Format(time.RFC3339)
	}
	if k.CreatedAt != nil {
		m["created_at"] = k.CreatedAt.Format(time.RFC3339)
	}
	return m
}
