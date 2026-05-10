package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	raff "github.com/rafftechnologies/raff-go"
	"github.com/rafftechnologies/raff-go/spec"
)

func resourceRole() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a custom IAM role. System roles (Owner, Admin, Member, Operator, etc.) are immutable and managed by the platform — only custom roles can be created here.",
		CreateContext: resourceRoleCreate,
		ReadContext:   resourceRoleRead,
		UpdateContext: resourceRoleUpdate,
		DeleteContext: resourceRoleDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Display name.",
			},
			"slug": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "URL-safe identifier.",
			},
			"scope": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "account or project.",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Role description.",
			},
			"permissions": {
				Type:        schema.TypeSet,
				Required:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Set of permission keys (e.g. vm.create, project.members.view). Run `raff permission list` for the catalog.",
			},
			// Computed
			"is_system":  {Type: schema.TypeBool, Computed: true},
			"created_at": {Type: schema.TypeString, Computed: true},
			"updated_at": {Type: schema.TypeString, Computed: true},
		},
	}
}

func resourceRoleCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)

	req := &raff.CreateRoleRequest{
		Name:        d.Get("name").(string),
		Slug:        d.Get("slug").(string),
		Scope:       spec.CreateRoleRequestScope(d.Get("scope").(string)),
		Permissions: expandStringSet(d.Get("permissions")),
	}
	if v := d.Get("description").(string); v != "" {
		req.Description = &v
	}
	r, _, err := client.Roles.Create(ctx, req)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(r.ID.String())
	return setRoleState(d, r)
}

func resourceRoleRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	r, _, err := client.Roles.Get(ctx, d.Id())
	if err != nil {
		if errResp, ok := err.(*raff.ErrorResponse); ok && errResp.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	return setRoleState(d, r)
}

func resourceRoleUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	req := &raff.UpdateRoleRequest{}
	if d.HasChange("name") {
		v := d.Get("name").(string)
		req.Name = &v
	}
	if d.HasChange("description") {
		v := d.Get("description").(string)
		req.Description = &v
	}
	if d.HasChange("permissions") {
		perms := expandStringSet(d.Get("permissions"))
		req.Permissions = &perms
	}
	r, _, err := client.Roles.Update(ctx, d.Id(), req)
	if err != nil {
		return diag.FromErr(err)
	}
	return setRoleState(d, r)
}

func resourceRoleDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	if _, err := client.Roles.Delete(ctx, d.Id()); err != nil {
		if errResp, ok := err.(*raff.ErrorResponse); ok && errResp.StatusCode == 404 {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}

func setRoleState(d *schema.ResourceData, r *raff.Role) diag.Diagnostics {
	d.Set("name", r.Name)
	d.Set("slug", r.Slug)
	d.Set("scope", string(r.Scope))
	d.Set("is_system", r.IsSystem)
	d.Set("permissions", r.Permissions)
	if r.Description != nil {
		d.Set("description", *r.Description)
	}
	if r.CreatedAt != nil {
		d.Set("created_at", r.CreatedAt.Format("2006-01-02T15:04:05Z"))
	}
	if r.UpdatedAt != nil {
		d.Set("updated_at", r.UpdatedAt.Format("2006-01-02T15:04:05Z"))
	}
	return nil
}

// expandStringSet converts a Terraform schema.Set of strings to []string.
func expandStringSet(v any) []string {
	set, ok := v.(*schema.Set)
	if !ok {
		return nil
	}
	out := make([]string, 0, set.Len())
	for _, item := range set.List() {
		out = append(out, item.(string))
	}
	return out
}
