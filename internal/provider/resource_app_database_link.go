package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	raff "github.com/rafftechnologies/raff-go"
)

func resourceAppDatabaseLink() *schema.Resource {
	return &schema.Resource{
		Description: "Links a managed database to a Raff Apps service. The app gets its own database user and DATABASE_URL, DATABASE_HOST, DATABASE_PORT, DATABASE_USER, DATABASE_PASSWORD, DATABASE_NAME and DATABASE_CA_CERT, then redeploys. Deleting the link removes the user and the variables.",

		CreateContext: resourceAppDatabaseLinkCreate,
		ReadContext:   resourceAppDatabaseLinkRead,
		DeleteContext: resourceAppDatabaseLinkDelete,

		Importer: &schema.ResourceImporter{
			StateContext: resourceAppDatabaseLinkImport,
		},

		Schema: map[string]*schema.Schema{
			"service_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "ID of the app (`raff_app_service.<name>.id`).",
			},
			"database_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "ID of a running managed database (`raff_database.<name>.id`).",
			},
			"env_prefix": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Prefix of the variables (default `DATABASE`). Give a second database its own prefix.",
				StateFunc:   func(v any) string { return strings.ToUpper(v.(string)) },
			},
			"database_name": {Type: schema.TypeString, Computed: true},
			"injected_keys": {
				Type:        schema.TypeList,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "The variables the link sets.",
			},
			"status":         {Type: schema.TypeString, Computed: true},
			"status_message": {Type: schema.TypeString, Computed: true},
		},
	}
}

func resourceAppDatabaseLinkCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	serviceID := d.Get("service_id").(string)
	link, _, err := client.AppServices.CreateBinding(ctx, serviceID, &raff.CreateAppBindingRequest{
		BindingType: "database",
		ResourceID:  d.Get("database_id").(string),
		EnvPrefix:   d.Get("env_prefix").(string),
	})
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(serviceID + "/" + link.ID)
	return resourceAppDatabaseLinkRead(ctx, d, meta)
}

func resourceAppDatabaseLinkRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	serviceID, linkID, err := splitAppChildID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	links, _, err := client.AppServices.ListBindings(ctx, serviceID)
	if err != nil {
		if errResp, ok := err.(*raff.ErrorResponse); ok && errResp.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	for _, l := range links {
		if l.ID == linkID {
			d.Set("service_id", serviceID)
			// The configuration may name the database by its short ID or its
			// UUID; the API answers with the UUID, so only an import sets it.
			if d.Get("database_id").(string) == "" {
				d.Set("database_id", l.ResourceID)
			}
			d.Set("env_prefix", l.EnvPrefix)
			d.Set("database_name", l.ResourceName)
			d.Set("injected_keys", l.InjectedKeys)
			d.Set("status", l.Status)
			d.Set("status_message", l.StatusMessage)
			return nil
		}
	}
	d.SetId("")
	return nil
}

func resourceAppDatabaseLinkDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	_, linkID, err := splitAppChildID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if _, err := client.AppServices.DeleteBinding(ctx, linkID); err != nil {
		if errResp, ok := err.(*raff.ErrorResponse); ok && errResp.StatusCode == 404 {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}

func resourceAppDatabaseLinkImport(ctx context.Context, d *schema.ResourceData, meta any) ([]*schema.ResourceData, error) {
	if _, _, err := splitAppChildID(d.Id()); err != nil {
		return nil, err
	}
	return []*schema.ResourceData{d}, nil
}
