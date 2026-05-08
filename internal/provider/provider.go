package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	raff "github.com/rafftechnologies/raff-go"
)

// New returns the Raff provider.
func New() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"api_key": {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("RAFF_API_KEY", nil),
				Description: "Raff API key (raff_pub_xxx). Can also be set via RAFF_API_KEY env var.",
			},
			"api_url": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("RAFF_API_URL", "https://api.rafftechnologies.com"),
				Description: "Raff API base URL. Can also be set via RAFF_API_URL env var.",
			},
			"project_id": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("RAFF_PROJECT_ID", nil),
				Description: "Default project ID for resources. Can also be set via RAFF_PROJECT_ID env var.",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"raff_project": resourceProject(),
			"raff_vm":      resourceVM(),
		},

		ConfigureContextFunc: configure,
	}
}

func configure(ctx context.Context, d *schema.ResourceData) (any, diag.Diagnostics) {
	apiKey := d.Get("api_key").(string)

	opts := []raff.ClientOpt{
		raff.SetUserAgent("terraform-provider-raff/0.1.0"),
	}

	if v, ok := d.GetOk("api_url"); ok {
		opts = append(opts, raff.SetBaseURL(v.(string)))
	}

	if v, ok := d.GetOk("project_id"); ok {
		opts = append(opts, raff.SetProjectID(v.(string)))
	}

	client := raff.NewFromToken(apiKey, opts...)

	return client, nil
}
