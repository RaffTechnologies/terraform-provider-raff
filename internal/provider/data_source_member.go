package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	raff "github.com/rafftechnologies/raff-go"
)

func memberComputedSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"id":         {Type: schema.TypeString, Computed: true},
		"email":      {Type: schema.TypeString, Computed: true},
		"role_id":    {Type: schema.TypeString, Computed: true},
		"role_name":  {Type: schema.TypeString, Computed: true},
		"status":     {Type: schema.TypeString, Computed: true},
		"user_id":    {Type: schema.TypeString, Computed: true},
		"created_at": {Type: schema.TypeString, Computed: true},
	}
}

func dataSourceMember() *schema.Resource {
	s := memberComputedSchema()
	s["id"] = &schema.Schema{Type: schema.TypeString, Required: true}
	return &schema.Resource{
		Description: "Reads a single account member by ID.",
		ReadContext: dataSourceMemberRead,
		Schema:      s,
	}
}

func dataSourceMemberRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	m, _, err := client.Members.Get(ctx, d.Get("id").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(m.ID.String())
	for k, v := range memberToMap(m) {
		d.Set(k, v)
	}
	return nil
}

func dataSourceMembers() *schema.Resource {
	return &schema.Resource{
		Description: "Lists account members.",
		ReadContext: dataSourceMembersRead,
		Schema: map[string]*schema.Schema{
			"members": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Resource{Schema: memberComputedSchema()},
			},
		},
	}
}

func dataSourceMembersRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	members, _, err := client.Members.List(ctx, nil)
	if err != nil {
		return diag.FromErr(err)
	}
	out := make([]map[string]any, 0, len(members))
	for i := range members {
		out = append(out, memberToMap(&members[i]))
	}
	d.SetId("members")
	d.Set("members", out)
	return nil
}

func memberToMap(m *raff.Member) map[string]any {
	mm := map[string]any{
		"id":     m.ID.String(),
		"email":  string(m.Email),
		"status": string(m.Status),
	}
	if m.RoleID != nil {
		mm["role_id"] = m.RoleID.String()
	}
	if m.RoleName != nil {
		mm["role_name"] = *m.RoleName
	}
	if m.UserID != nil {
		mm["user_id"] = m.UserID.String()
	}
	if m.CreatedAt != nil {
		mm["created_at"] = m.CreatedAt.Format("2006-01-02T15:04:05Z")
	}
	return mm
}
