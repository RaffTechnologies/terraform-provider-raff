package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	raff "github.com/rafftechnologies/raff-go"
	"github.com/rafftechnologies/raff-go/spec"
)

// dataSourcePermissions exposes the permission catalog so role construction
// can reference permission names declaratively (e.g. `permissions = data.raff_permissions.project.permissions[*].name`)
// instead of hardcoding strings in `raff_role`.
func dataSourcePermissions() *schema.Resource {
	return &schema.Resource{
		Description: "Reads the read-only permission catalog. Use the `name` field as a permission identifier on `raff_role`.",
		ReadContext: dataSourcePermissionsRead,
		Schema: map[string]*schema.Schema{
			"scope": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Filter by scope: `account` or `project`.",
			},
			"permissions": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{Schema: map[string]*schema.Schema{
					"name":        {Type: schema.TypeString, Computed: true},
					"category":    {Type: schema.TypeString, Computed: true},
					"description": {Type: schema.TypeString, Computed: true},
					"scope":       {Type: schema.TypeString, Computed: true},
				}},
			},
		},
	}
}

func dataSourcePermissionsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)

	opts := &raff.PermissionListOptions{}
	if v, ok := d.GetOk("scope"); ok {
		s := spec.ListPermissionsParamsScope(v.(string))
		opts.Scope = &s
	}

	perms, _, err := client.Permissions.List(ctx, opts)
	if err != nil {
		return diag.FromErr(err)
	}

	out := make([]map[string]any, 0, len(perms))
	for _, p := range perms {
		out = append(out, map[string]any{
			"name":        p.Name,
			"category":    p.Category,
			"description": raff.StringValue(p.Description),
			"scope":       string(p.Scope),
		})
	}

	id := "permissions_all"
	if v, ok := d.GetOk("scope"); ok {
		id = "permissions_" + v.(string)
	}
	d.SetId(id)
	d.Set("permissions", out)
	return nil
}
