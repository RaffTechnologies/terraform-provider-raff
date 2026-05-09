package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	raff "github.com/rafftechnologies/raff-go"
)

func projectComputedSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"id":             {Type: schema.TypeString, Computed: true},
		"name":           {Type: schema.TypeString, Computed: true},
		"description":    {Type: schema.TypeString, Computed: true},
		"slug":           {Type: schema.TypeString, Computed: true},
		"default_region": {Type: schema.TypeString, Computed: true},
		"account_id":     {Type: schema.TypeString, Computed: true},
		"is_default":     {Type: schema.TypeBool, Computed: true},
		"is_active":      {Type: schema.TypeBool, Computed: true},
		"created_by":     {Type: schema.TypeString, Computed: true},
		"created_at":     {Type: schema.TypeString, Computed: true},
		"updated_at":     {Type: schema.TypeString, Computed: true},
	}
}

func dataSourceProject() *schema.Resource {
	s := projectComputedSchema()
	s["id"] = &schema.Schema{Type: schema.TypeString, Required: true}
	return &schema.Resource{
		Description: "Reads a single Raff project by ID.",
		ReadContext: dataSourceProjectRead,
		Schema:      s,
	}
}

func dataSourceProjectRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	p, _, err := client.Projects.Get(ctx, d.Get("id").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(p.ID.String())
	for k, v := range projectToMap(p) {
		d.Set(k, v)
	}
	return nil
}

func dataSourceProjects() *schema.Resource {
	return &schema.Resource{
		Description: "Lists Raff projects accessible to the current API key.",
		ReadContext: dataSourceProjectsRead,
		Schema: map[string]*schema.Schema{
			"projects": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Resource{Schema: projectComputedSchema()},
			},
		},
	}
}

func dataSourceProjectsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	projects, _, err := client.Projects.List(ctx, nil)
	if err != nil {
		return diag.FromErr(err)
	}
	out := make([]map[string]any, 0, len(projects))
	for i := range projects {
		out = append(out, projectToMap(&projects[i]))
	}
	d.SetId("projects")
	d.Set("projects", out)
	return nil
}

func projectToMap(p *raff.Project) map[string]any {
	m := map[string]any{
		"id":             p.ID.String(),
		"name":           p.Name,
		"description":    raff.StringValue(p.Description),
		"slug":           p.Slug,
		"default_region": string(p.DefaultRegion),
		"account_id":     p.AccountID.String(),
		"is_default":     p.IsDefault,
		"is_active":      p.IsActive,
		"created_at":     p.CreatedAt.Format("2006-01-02T15:04:05Z"),
		"updated_at":     p.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if p.CreatedBy != nil {
		m["created_by"] = p.CreatedBy.String()
	} else {
		m["created_by"] = ""
	}
	return m
}
