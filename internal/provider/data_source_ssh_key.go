package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	raff "github.com/rafftechnologies/raff-go"
)

func sshKeyComputedSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"id":         {Type: schema.TypeString, Computed: true},
		"name":       {Type: schema.TypeString, Computed: true},
		"public_key": {Type: schema.TypeString, Computed: true},
		"key_type":   {Type: schema.TypeString, Computed: true},
		"created_at": {Type: schema.TypeString, Computed: true},
		"updated_at": {Type: schema.TypeString, Computed: true},
	}
}

func dataSourceSSHKey() *schema.Resource {
	s := sshKeyComputedSchema()
	s["id"] = &schema.Schema{Type: schema.TypeString, Required: true}
	return &schema.Resource{
		Description: "Reads a single SSH key by ID.",
		ReadContext: dataSourceSSHKeyRead,
		Schema:      s,
	}
}

func dataSourceSSHKeyRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	k, _, err := client.SSHKeys.Get(ctx, d.Get("id").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(k.ID.String())
	for kk, vv := range sshKeyToMap(k) {
		d.Set(kk, vv)
	}
	return nil
}

func dataSourceSSHKeys() *schema.Resource {
	return &schema.Resource{
		Description: "Lists all SSH keys for the account.",
		ReadContext: dataSourceSSHKeysRead,
		Schema: map[string]*schema.Schema{
			"ssh_keys": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Resource{Schema: sshKeyComputedSchema()},
			},
		},
	}
}

func dataSourceSSHKeysRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	keys, _, err := client.SSHKeys.List(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	out := make([]map[string]any, 0, len(keys))
	for i := range keys {
		out = append(out, sshKeyToMap(&keys[i]))
	}
	d.SetId("ssh_keys")
	d.Set("ssh_keys", out)
	return nil
}

func sshKeyToMap(k *raff.SSHKey) map[string]any {
	m := map[string]any{
		"id":         k.ID.String(),
		"name":       k.Name,
		"public_key": k.PublicKey,
		"key_type":   string(k.KeyType),
	}
	if k.CreatedAt != nil {
		m["created_at"] = k.CreatedAt.Format("2006-01-02T15:04:05Z")
	}
	if k.UpdatedAt != nil {
		m["updated_at"] = k.UpdatedAt.Format("2006-01-02T15:04:05Z")
	}
	return m
}
