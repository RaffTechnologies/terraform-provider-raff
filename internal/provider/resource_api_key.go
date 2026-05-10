package provider

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	raff "github.com/rafftechnologies/raff-go"
	"github.com/rafftechnologies/raff-go/spec"
)

func resourceAPIKey() *schema.Resource {
	return &schema.Resource{
		Description: `Creates a Raff API key. The plaintext secret is returned ONCE at create time and stored in the secret attribute (sensitive). Lock down your Terraform state — anyone with read access to it can use the key.`,
		CreateContext: resourceAPIKeyCreate,
		ReadContext:   resourceAPIKeyRead,
		UpdateContext: resourceAPIKeyUpdate,
		DeleteContext: resourceAPIKeyDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Display name for the key.",
			},
			"rate_limit_tier": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "standard (default, 30 RPS) or high (100 RPS, requires support approval).",
			},
			"expires_at": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Expiration in RFC3339 (e.g. 2026-12-31T23:59:59Z). Omit for never-expires.",
			},
			"is_active": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Whether the key is active. Set to false to suspend without revoking.",
			},
			// Computed
			"key_prefix": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "First 13 characters of the key (e.g. raff_pub_17d70fcf).",
			},
			"secret": {
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "Full plaintext API key. Returned only once on create or regenerate; never re-fetchable.",
			},
			"created_at": {Type: schema.TypeString, Computed: true},
		},
	}
}

func resourceAPIKeyCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)

	req := &raff.CreateAPIKeyRequest{Name: d.Get("name").(string)}
	if v, ok := d.GetOk("rate_limit_tier"); ok {
		t := spec.CreateAPIKeyRequestRateLimitTier(v.(string))
		req.RateLimitTier = &t
	}
	if v, ok := d.GetOk("expires_at"); ok {
		t, err := time.Parse(time.RFC3339, v.(string))
		if err != nil {
			return diag.Errorf("expires_at must be RFC3339: %s", err)
		}
		req.ExpiresAt = &t
	}

	k, _, err := client.APIKeys.Create(ctx, req)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(k.ID.String())
	d.Set("secret", k.Secret) // only available on create — store in state
	return setAPIKeyStateFromWithSecret(d, k)
}

func resourceAPIKeyRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	k, _, err := client.APIKeys.Get(ctx, d.Id())
	if err != nil {
		if errResp, ok := err.(*raff.ErrorResponse); ok && errResp.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	return setAPIKeyState(d, k)
}

func resourceAPIKeyUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	req := &raff.UpdateAPIKeyRequest{}
	changed := false
	if d.HasChange("name") {
		v := d.Get("name").(string)
		req.Name = &v
		changed = true
	}
	if d.HasChange("rate_limit_tier") {
		t := spec.UpdateAPIKeyRequestRateLimitTier(d.Get("rate_limit_tier").(string))
		req.RateLimitTier = &t
		changed = true
	}
	if d.HasChange("is_active") {
		v := d.Get("is_active").(bool)
		req.IsActive = &v
		changed = true
	}
	if d.HasChange("expires_at") {
		v := d.Get("expires_at").(string)
		if v != "" {
			t, err := time.Parse(time.RFC3339, v)
			if err != nil {
				return diag.Errorf("expires_at must be RFC3339: %s", err)
			}
			req.ExpiresAt = &t
		}
		changed = true
	}
	if changed {
		if _, _, err := client.APIKeys.Update(ctx, d.Id(), req); err != nil {
			return diag.FromErr(err)
		}
	}
	return resourceAPIKeyRead(ctx, d, meta)
}

func resourceAPIKeyDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	if _, err := client.APIKeys.Revoke(ctx, d.Id()); err != nil {
		if errResp, ok := err.(*raff.ErrorResponse); ok && errResp.StatusCode == 404 {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}

func setAPIKeyState(d *schema.ResourceData, k *raff.APIKey) diag.Diagnostics {
	d.Set("name", k.Name)
	d.Set("key_prefix", k.KeyPrefix)
	d.Set("is_active", k.IsActive)
	if k.RateLimitTier != nil {
		d.Set("rate_limit_tier", string(*k.RateLimitTier))
	}
	if k.ExpiresAt != nil {
		d.Set("expires_at", k.ExpiresAt.Format(time.RFC3339))
	}
	if k.CreatedAt != nil {
		d.Set("created_at", k.CreatedAt.Format(time.RFC3339))
	}
	return nil
}

func setAPIKeyStateFromWithSecret(d *schema.ResourceData, k *raff.APIKeyWithSecret) diag.Diagnostics {
	d.Set("name", k.Name)
	d.Set("key_prefix", k.KeyPrefix)
	d.Set("is_active", k.IsActive)
	if k.RateLimitTier != nil {
		d.Set("rate_limit_tier", string(*k.RateLimitTier))
	}
	if k.ExpiresAt != nil {
		d.Set("expires_at", k.ExpiresAt.Format(time.RFC3339))
	}
	if k.CreatedAt != nil {
		d.Set("created_at", k.CreatedAt.Format(time.RFC3339))
	}
	return nil
}
