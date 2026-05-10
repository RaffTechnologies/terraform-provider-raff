package provider

import (
	"context"

	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	raff "github.com/rafftechnologies/raff-go"
	"github.com/rafftechnologies/raff-go/spec"
)

func resourceMember() *schema.Resource {
	return &schema.Resource{
		Description: `Manages an account-level member. Use one of (email, target_user_id, api_key_id) to identify the member at create time — they're mutually exclusive.

Inviting by email creates a pending member that becomes active when the recipient accepts the invitation.`,
		CreateContext: resourceMemberCreate,
		ReadContext:   resourceMemberRead,
		UpdateContext: resourceMemberUpdate,
		DeleteContext: resourceMemberDelete,

		Schema: map[string]*schema.Schema{
			"email": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{"email", "target_user_id", "api_key_id"},
				Description:  "Email of the user to invite. Mutually exclusive with target_user_id and api_key_id.",
			},
			"target_user_id": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Existing user UUID to add directly. Mutually exclusive with email and api_key_id.",
			},
			"api_key_id": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "API key UUID to grant account-level access. Mutually exclusive with email and target_user_id.",
			},
			"role_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Role UUID. Must be account-scoped.",
			},
			"status": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "active or suspended. pending is set automatically for invited members until they accept.",
			},
			// Computed
			"role_name":  {Type: schema.TypeString, Computed: true},
			"created_at": {Type: schema.TypeString, Computed: true},
		},
	}
}

func resourceMemberCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	roleID, err := uuid.Parse(d.Get("role_id").(string))
	if err != nil {
		return diag.Errorf("invalid role_id: %s", err)
	}
	req := &raff.AddMemberRequest{RoleID: &roleID}
	if v, ok := d.GetOk("email"); ok {
		e := openapi_types.Email(v.(string))
		req.Email = &e
	}
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
	m, _, err := client.Members.Add(ctx, req)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(m.ID.String())
	return setMemberState(d, m)
}

func resourceMemberRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	m, _, err := client.Members.Get(ctx, d.Id())
	if err != nil {
		if errResp, ok := err.(*raff.ErrorResponse); ok && errResp.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	return setMemberState(d, m)
}

func resourceMemberUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	req := &raff.UpdateMemberRequest{}
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
	m, _, err := client.Members.Update(ctx, d.Id(), req)
	if err != nil {
		return diag.FromErr(err)
	}
	return setMemberState(d, m)
}

func resourceMemberDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	if _, err := client.Members.Remove(ctx, d.Id()); err != nil {
		if errResp, ok := err.(*raff.ErrorResponse); ok && errResp.StatusCode == 404 {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}

func setMemberState(d *schema.ResourceData, m *raff.Member) diag.Diagnostics {
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
