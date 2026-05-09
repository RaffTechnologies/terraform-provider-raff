package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	raff "github.com/rafftechnologies/raff-go"
	"github.com/rafftechnologies/raff-go/spec"
)

func resourceIP() *schema.Resource {
	return &schema.Resource{
		Description:   "Reserves a Raff floating IP. The IP is held for the project until the resource is destroyed.",
		CreateContext: resourceIPCreate,
		ReadContext:   resourceIPRead,
		DeleteContext: resourceIPDelete,

		Schema: map[string]*schema.Schema{
			"type": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Default:     "ipv4",
				Description: "IP family to reserve (ipv4 or ipv6). Defaults to ipv4.",
			},
			"region": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Region for the reserved IP. Defaults to the project's default region.",
			},
			"billing_period": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Subscription billing period. Ignored for PAYG accounts.",
			},
			// Computed
			"ip_address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The reserved IP address.",
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Whether the IP is free or in-use.",
			},
			"reserved": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"account_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"project_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"created_at": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"updated_at": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceIPCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)

	req := &raff.ReserveIPRequest{}
	if v, ok := d.GetOk("type"); ok {
		t := spec.ReserveIPRequestType(v.(string))
		req.Type = &t
	}
	if v, ok := d.GetOk("region"); ok {
		r := spec.ReserveIPRequestRegion(v.(string))
		req.Region = &r
	}
	if v, ok := d.GetOk("billing_period"); ok {
		bp := spec.ReserveIPRequestBillingPeriod(v.(string))
		req.BillingPeriod = &bp
	}

	ip, _, err := client.IPs.Reserve(ctx, req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(ip.ID.String())

	return setIPState(d, ip)
}

func resourceIPRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)

	ip, _, err := client.IPs.Get(ctx, d.Id())
	if err != nil {
		if errResp, ok := err.(*raff.ErrorResponse); ok && errResp.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	return setIPState(d, ip)
}

func resourceIPDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)

	_, err := client.IPs.Release(ctx, d.Id())
	if err != nil {
		if errResp, ok := err.(*raff.ErrorResponse); ok && errResp.StatusCode == 404 {
			return nil
		}
		return diag.FromErr(err)
	}

	return nil
}

func setIPState(d *schema.ResourceData, ip *raff.FloatingIP) diag.Diagnostics {
	d.Set("ip_address", ip.IPAddress)
	d.Set("status", string(ip.Status))
	d.Set("reserved", ip.Reserved)
	d.Set("type", string(ip.Type))
	if ip.Region != nil {
		d.Set("region", string(*ip.Region))
	}
	if ip.AccountID != nil {
		d.Set("account_id", ip.AccountID.String())
	}
	if ip.ProjectID != nil {
		d.Set("project_id", ip.ProjectID.String())
	}
	if ip.CreatedAt != nil {
		d.Set("created_at", ip.CreatedAt.Format("2006-01-02T15:04:05Z"))
	}
	if ip.UpdatedAt != nil {
		d.Set("updated_at", ip.UpdatedAt.Format("2006-01-02T15:04:05Z"))
	}

	return nil
}
