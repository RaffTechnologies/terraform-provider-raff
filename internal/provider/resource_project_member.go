package provider

import (
	"context"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	raff "github.com/rafftechnologies/raff-go"
	"github.com/rafftechnologies/raff-go/spec"
)

func resourceProjectMember() *schema.Resource {
	return &schema.Resource{
		Description:   "Adds an existing account user (or API key) to a project with a project-scoped role. Account-level invitations should use raff_member instead.",
		CreateContext: resourceProjectMemberCreate,
		ReadContext:   resourceProjectMemberRead,
		UpdateContext: resourceProjectMemberUpdate,
		DeleteContext: resourceProjectMemberDelete,

		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Project UUID.",
			},
			"target_user_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{"target_user_id", "api_key_id"},
				Description:  "Existing account user UUID. Mutually exclusive with api_key_id.",
			},
			"api_key_id": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "API key UUID. Mutually exclusive with target_user_id.",
			},
			"role_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Role UUID. Must be project-scoped.",
			},
			"status": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "active or suspended.",
			},
			// Computed
			"email":      {Type: schema.TypeString, Computed: true},
			"role_name":  {Type: schema.TypeString, Computed: true},
			"created_at": {Type: schema.TypeString, Computed: true},
		},
	}
}

func resourceProjectMemberCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	projectID := d.Get("project_id").(string)
	roleID, err := uuid.Parse(d.Get("role_id").(string))
	if err != nil {
		return diag.Errorf("invalid role_id: %s", err)
	}
	req := &raff.AddProjectMemberRequest{RoleID: roleID}
	if v, ok := d.GetOk("target_user_id"); ok {
		uid, err := uuid.Parse(v.(string))
		if err != nil {
			return diag.Errorf("invalid target_user_id: %s", err)
		}
		req.TargetUserID = &uid
	}
	if v, ok := d.GetOk("api_key_id"); ok {
		kid, err := uuid.Parse(v.(string))
		if err != nil {
			return diag.Errorf("invalid api_key_id: %s", err)
		}
		req.APIKeyID = &kid
	}
	m, _, err := client.ProjectMembers.Add(ctx, projectID, req)
	if err != nil {
		return diag.FromErr(err)
	}
	// Composite ID: <project_id>/<member_id> so Read can route correctly.
	d.SetId(projectID + "/" + m.ID.String())
	return setProjectMemberState(d, m)
}

func resourceProjectMemberRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	projectID, memberID, err := splitProjectMemberID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	m, _, err := client.ProjectMembers.Get(ctx, projectID, memberID)
	if err != nil {
		if errResp, ok := err.(*raff.ErrorResponse); ok && errResp.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	return setProjectMemberState(d, m)
}

func resourceProjectMemberUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	projectID, memberID, err := splitProjectMemberID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	req := &raff.UpdateProjectMemberRequest{}
	if d.HasChange("role_id") {
		rid, err := uuid.Parse(d.Get("role_id").(string))
		if err != nil {
			return diag.Errorf("invalid role_id: %s", err)
		}
		req.RoleID = &rid
	}
	if d.HasChange("status") {
		s := spec.UpdateMemberRequestStatus(d.Get("status").(string))
		req.Status = &s
	}
	m, _, err := client.ProjectMembers.Update(ctx, projectID, memberID, req)
	if err != nil {
		return diag.FromErr(err)
	}
	return setProjectMemberState(d, m)
}

func resourceProjectMemberDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	projectID, memberID, err := splitProjectMemberID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if _, err := client.ProjectMembers.Remove(ctx, projectID, memberID); err != nil {
		if errResp, ok := err.(*raff.ErrorResponse); ok && errResp.StatusCode == 404 {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}

func setProjectMemberState(d *schema.ResourceData, m *raff.ProjectMember) diag.Diagnostics {
	d.Set("email", string(m.Email))
	d.Set("status", string(m.Status))
	if m.RoleID != nil {
		d.Set("role_id", m.RoleID.String())
	}
	if m.RoleName != nil {
		d.Set("role_name", *m.RoleName)
	}
	if m.CreatedAt != nil {
		d.Set("created_at", m.CreatedAt.Format("2006-01-02T15:04:05Z"))
	}
	return nil
}

// splitProjectMemberID splits a "<project_id>/<member_id>" composite ID.
func splitProjectMemberID(id string) (string, string, error) {
	for i := 0; i < len(id); i++ {
		if id[i] == '/' {
			return id[:i], id[i+1:], nil
		}
	}
	return "", "", &compositeIDError{id: id}
}

type compositeIDError struct{ id string }

func (e *compositeIDError) Error() string {
	return "invalid project member ID " + e.id + ": expected <project_id>/<member_id>"
}
