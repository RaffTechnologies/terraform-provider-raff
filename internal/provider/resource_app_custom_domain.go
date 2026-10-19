package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	raff "github.com/rafftechnologies/raff-go"
)

func resourceAppCustomDomain() *schema.Resource {
	return &schema.Resource{
		Description: "Adds a custom domain to a Raff Apps web service. Create the CNAME and TXT records it returns; verification and the TLS certificate then run automatically.",

		CreateContext: resourceAppCustomDomainCreate,
		ReadContext:   resourceAppCustomDomainRead,
		DeleteContext: resourceAppCustomDomainDelete,

		Importer: &schema.ResourceImporter{
			StateContext: resourceAppCustomDomainImport,
		},

		Schema: map[string]*schema.Schema{
			"service_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "ID of the web app (`raff_app_service.<name>.id`).",
			},
			"domain": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The hostname, such as `www.example.com`.",
			},
			"status":                 {Type: schema.TypeString, Computed: true, Description: "Verification and certificate status."},
			"status_message":         {Type: schema.TypeString, Computed: true},
			"cname_target":           {Type: schema.TypeString, Computed: true, Description: "Point the domain's CNAME record here."},
			"verification_txt_name":  {Type: schema.TypeString, Computed: true, Description: "Name of the TXT record that proves you own the domain."},
			"verification_txt_value": {Type: schema.TypeString, Computed: true, Description: "Value of that TXT record."},
		},
	}
}

func resourceAppCustomDomainCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	serviceID := d.Get("service_id").(string)
	dom, _, err := client.AppServices.AddCustomDomain(ctx, serviceID, &raff.AddAppCustomDomainRequest{Domain: d.Get("domain").(string)})
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(serviceID + "/" + dom.ID)
	return resourceAppCustomDomainRead(ctx, d, meta)
}

func resourceAppCustomDomainRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	serviceID, domainID, err := splitAppChildID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	domains, _, err := client.AppServices.ListCustomDomains(ctx, serviceID)
	if err != nil {
		if errResp, ok := err.(*raff.ErrorResponse); ok && errResp.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	for _, dom := range domains {
		if dom.ID == domainID {
			d.Set("service_id", serviceID)
			d.Set("domain", dom.Domain)
			d.Set("status", dom.Status)
			d.Set("status_message", dom.StatusMessage)
			d.Set("cname_target", dom.CNAMETarget)
			d.Set("verification_txt_name", dom.VerificationTXTName)
			d.Set("verification_txt_value", dom.VerificationTXTValue)
			return nil
		}
	}
	d.SetId("")
	return nil
}

func resourceAppCustomDomainDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	_, domainID, err := splitAppChildID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if _, err := client.AppServices.DeleteCustomDomain(ctx, domainID); err != nil {
		if errResp, ok := err.(*raff.ErrorResponse); ok && errResp.StatusCode == 404 {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}

func resourceAppCustomDomainImport(ctx context.Context, d *schema.ResourceData, meta any) ([]*schema.ResourceData, error) {
	if _, _, err := splitAppChildID(d.Id()); err != nil {
		return nil, err
	}
	return []*schema.ResourceData{d}, nil
}

// splitAppChildID splits "<service id>/<child id>", the ID of a resource that
// belongs to an app (a custom domain or a database link).
func splitAppChildID(id string) (string, string, error) {
	serviceID, childID, ok := strings.Cut(id, "/")
	if !ok || serviceID == "" || childID == "" {
		return "", "", fmt.Errorf("invalid ID %q: use <service id>/<id>", id)
	}
	return serviceID, childID, nil
}
