package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	raff "github.com/rafftechnologies/raff-go"
	"github.com/rafftechnologies/raff-go/spec"
)

func resourceProject() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a Raff project.",
		CreateContext: resourceProjectCreate,
		ReadContext:   resourceProjectRead,
		UpdateContext: resourceProjectUpdate,
		DeleteContext: resourceProjectDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Project name.",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Project description.",
			},
			"default_region": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "us-east",
				Description: "Default region for resources in this project.",
			},
			// Computed fields (read-only)
			"account_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Account ID this project belongs to.",
			},
			"slug": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "URL-friendly project identifier.",
			},
			"is_default": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether this is the account's default project.",
			},
			"is_active": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the project is active.",
			},
			"created_by": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "User who created the project.",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "When the project was created.",
			},
			"updated_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "When the project was last updated.",
			},
		},
	}
}

func resourceProjectCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)

	req := &raff.CreateProjectRequest{Name: d.Get("name").(string)}
	if v := d.Get("description").(string); v != "" {
		req.Description = raff.String(v)
	}
	if v := d.Get("default_region").(string); v != "" {
		r := spec.CreateProjectRequestDefaultRegion(v)
		req.DefaultRegion = &r
	}

	project, _, err := client.Projects.Create(ctx, req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(project.ID.String())

	return setProjectState(d, project)
}

func resourceProjectRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)

	project, _, err := client.Projects.Get(ctx, d.Id())
	if err != nil {
		if errResp, ok := err.(*raff.ErrorResponse); ok && errResp.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	return setProjectState(d, project)
}

func resourceProjectUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)

	req := &raff.UpdateProjectRequest{}
	if d.HasChange("name") {
		req.Name = raff.String(d.Get("name").(string))
	}
	if d.HasChange("description") {
		req.Description = raff.String(d.Get("description").(string))
	}
	if d.HasChange("default_region") {
		r := spec.UpdateProjectRequestDefaultRegion(d.Get("default_region").(string))
		req.DefaultRegion = &r
	}

	project, _, err := client.Projects.Update(ctx, d.Id(), req)
	if err != nil {
		return diag.FromErr(err)
	}

	return setProjectState(d, project)
}

func resourceProjectDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)

	_, err := client.Projects.Delete(ctx, d.Id())
	if err != nil {
		if errResp, ok := err.(*raff.ErrorResponse); ok && errResp.StatusCode == 404 {
			return nil
		}
		return diag.FromErr(err)
	}

	return nil
}

func setProjectState(d *schema.ResourceData, p *raff.Project) diag.Diagnostics {
	d.Set("name", p.Name)
	d.Set("description", raff.StringValue(p.Description))
	d.Set("default_region", string(p.DefaultRegion))
	d.Set("account_id", p.AccountID.String())
	d.Set("slug", p.Slug)
	d.Set("is_default", p.IsDefault)
	d.Set("is_active", p.IsActive)
	if p.CreatedBy != nil {
		d.Set("created_by", p.CreatedBy.String())
	} else {
		d.Set("created_by", "")
	}
	d.Set("created_at", p.CreatedAt.Format("2006-01-02T15:04:05Z"))
	d.Set("updated_at", p.UpdatedAt.Format("2006-01-02T15:04:05Z"))

	return nil
}
