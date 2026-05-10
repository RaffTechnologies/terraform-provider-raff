package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	raff "github.com/rafftechnologies/raff-go"
)

func projectMemberComputedSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"id":         {Type: schema.TypeString, Computed: true},
		"email":      {Type: schema.TypeString, Computed: true},
		"role_id":    {Type: schema.TypeString, Computed: true},
		"role_name":  {Type: schema.TypeString, Computed: true},
		"status":     {Type: schema.TypeString, Computed: true},
		"created_at": {Type: schema.TypeString, Computed: true},
	}
}

func dataSourceProjectMembers() *schema.Resource {
	return &schema.Resource{
		Description: "Lists members of a project.",
		ReadContext: dataSourceProjectMembersRead,
		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Project UUID.",
			},
			"project_members": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Resource{Schema: projectMemberComputedSchema()},
			},
		},
	}
}

func dataSourceProjectMembersRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	projectID := d.Get("project_id").(string)
	members, _, err := client.ProjectMembers.List(ctx, projectID, nil)
	if err != nil {
		return diag.FromErr(err)
	}
	out := make([]map[string]any, 0, len(members))
	for i := range members {
		out = append(out, projectMemberToMap(&members[i]))
	}
	d.SetId("project_members/" + projectID)
	d.Set("project_members", out)
	return nil
}

func projectMemberToMap(m *raff.ProjectMember) map[string]any {
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
	if m.CreatedAt != nil {
		mm["created_at"] = m.CreatedAt.Format("2006-01-02T15:04:05Z")
	}
	return mm
}
