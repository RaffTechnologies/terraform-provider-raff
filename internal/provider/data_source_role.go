package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	raff "github.com/rafftechnologies/raff-go"
)

func roleComputedSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"id":          {Type: schema.TypeString, Computed: true},
		"name":        {Type: schema.TypeString, Computed: true},
		"slug":        {Type: schema.TypeString, Computed: true},
		"scope":       {Type: schema.TypeString, Computed: true},
		"description": {Type: schema.TypeString, Computed: true},
		"is_system":   {Type: schema.TypeBool, Computed: true},
		"permissions": {
			Type:     schema.TypeList,
			Computed: true,
			Elem:     &schema.Schema{Type: schema.TypeString},
		},
		"created_at": {Type: schema.TypeString, Computed: true},
		"updated_at": {Type: schema.TypeString, Computed: true},
	}
}

func dataSourceRole() *schema.Resource {
	s := roleComputedSchema()
	s["id"] = &schema.Schema{Type: schema.TypeString, Required: true}
	return &schema.Resource{
		Description: "Reads a single role by ID.",
		ReadContext: dataSourceRoleRead,
		Schema:      s,
	}
}

func dataSourceRoleRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	r, _, err := client.Roles.Get(ctx, d.Get("id").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(r.ID.String())
	for k, v := range roleToMap(r) {
		d.Set(k, v)
	}
	return nil
}

func dataSourceRoles() *schema.Resource {
	return &schema.Resource{
		Description: "Lists roles, optionally filtered by scope.",
		ReadContext: dataSourceRolesRead,
		Schema: map[string]*schema.Schema{
			"scope": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Filter: account or project.",
			},
			"roles": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Resource{Schema: roleComputedSchema()},
			},
		},
	}
}

func dataSourceRolesRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	roles, _, err := client.Roles.List(ctx, nil)
	if err != nil {
		return diag.FromErr(err)
	}
	scope, _ := d.GetOk("scope")
	out := make([]map[string]any, 0, len(roles))
	for i := range roles {
		if scope != nil && scope.(string) != "" && string(roles[i].Scope) != scope.(string) {
			continue
		}
		out = append(out, roleToMap(&roles[i]))
	}
	d.SetId("roles")
	d.Set("roles", out)
	return nil
}

func roleToMap(r *raff.Role) map[string]any {
	m := map[string]any{
		"id":          r.ID.String(),
		"name":        r.Name,
		"slug":        r.Slug,
		"scope":       string(r.Scope),
		"is_system":   r.IsSystem,
		"permissions": r.Permissions,
	}
	if r.Description != nil {
		m["description"] = *r.Description
	}
	if r.CreatedAt != nil {
		m["created_at"] = r.CreatedAt.Format("2006-01-02T15:04:05Z")
	}
	if r.UpdatedAt != nil {
		m["updated_at"] = r.UpdatedAt.Format("2006-01-02T15:04:05Z")
	}
	return m
}
