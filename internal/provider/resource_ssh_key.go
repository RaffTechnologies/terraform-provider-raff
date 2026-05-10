package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	raff "github.com/rafftechnologies/raff-go"
)

func resourceSSHKey() *schema.Resource {
	return &schema.Resource{
		Description:   "Registers an SSH public key for use when creating Linux VMs.",
		CreateContext: resourceSSHKeyCreate,
		ReadContext:   resourceSSHKeyRead,
		UpdateContext: resourceSSHKeyUpdate,
		DeleteContext: resourceSSHKeyDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Display name.",
			},
			"public_key": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Full SSH public key string (e.g. \"ssh-ed25519 AAAA... user@host\").",
			},
			// Computed
			"key_type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Parsed algorithm (ed25519, rsa, etc.).",
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

func resourceSSHKeyCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)

	k, _, err := client.SSHKeys.Create(ctx, &raff.CreateSSHKeyRequest{
		Name:      d.Get("name").(string),
		PublicKey: d.Get("public_key").(string),
	})
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(k.ID.String())
	return setSSHKeyState(d, k)
}

func resourceSSHKeyRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	k, _, err := client.SSHKeys.Get(ctx, d.Id())
	if err != nil {
		if errResp, ok := err.(*raff.ErrorResponse); ok && errResp.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	return setSSHKeyState(d, k)
}

func resourceSSHKeyUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	if d.HasChange("name") {
		k, _, err := client.SSHKeys.Update(ctx, d.Id(), &raff.UpdateSSHKeyRequest{Name: d.Get("name").(string)})
		if err != nil {
			return diag.FromErr(err)
		}
		return setSSHKeyState(d, k)
	}
	return resourceSSHKeyRead(ctx, d, meta)
}

func resourceSSHKeyDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	if _, err := client.SSHKeys.Delete(ctx, d.Id()); err != nil {
		if errResp, ok := err.(*raff.ErrorResponse); ok && errResp.StatusCode == 404 {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}

func setSSHKeyState(d *schema.ResourceData, k *raff.SSHKey) diag.Diagnostics {
	d.Set("name", k.Name)
	d.Set("public_key", k.PublicKey)
	d.Set("key_type", string(k.KeyType))
	if k.CreatedAt != nil {
		d.Set("created_at", k.CreatedAt.Format("2006-01-02T15:04:05Z"))
	}
	if k.UpdatedAt != nil {
		d.Set("updated_at", k.UpdatedAt.Format("2006-01-02T15:04:05Z"))
	}
	return nil
}
