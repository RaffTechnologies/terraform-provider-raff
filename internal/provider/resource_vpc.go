package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	raff "github.com/rafftechnologies/raff-go"
	"github.com/rafftechnologies/raff-go/spec"
)

func resourceVPC() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a Raff VPC.",
		CreateContext: resourceVPCCreate,
		ReadContext:   resourceVPCRead,
		UpdateContext: resourceVPCUpdate,
		DeleteContext: resourceVPCDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "VPC name.",
			},
			"cidr": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "CIDR block. Cannot be changed after creation.",
			},
			"region": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Default:     "us-east",
				Description: "VPC region.",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "VPC description.",
			},
			// Computed
			"account_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"project_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"gateway": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"gateway_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"dns": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"router_public_ip": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"router_status": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"total_ips": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"used_ips": {
				Type:     schema.TypeInt,
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

func resourceVPCCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)

	req := &raff.CreateVPCRequest{
		Name: d.Get("name").(string),
		Cidr: d.Get("cidr").(string),
	}
	if v, ok := d.GetOk("region"); ok {
		r := spec.CreateVPCRequestRegion(v.(string))
		req.Region = &r
	}

	vpc, _, err := client.VPCs.Create(ctx, req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(vpc.ID.String())

	// Description is set via Update — Create does not accept it.
	if v := d.Get("description").(string); v != "" {
		updReq := &raff.UpdateVPCRequest{Description: raff.String(v)}
		if vpc, _, err = client.VPCs.Update(ctx, vpc.ID.String(), updReq); err != nil {
			return diag.FromErr(err)
		}
	}

	return setVPCState(d, vpc)
}

func resourceVPCRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)

	vpc, _, err := client.VPCs.Get(ctx, d.Id())
	if err != nil {
		if errResp, ok := err.(*raff.ErrorResponse); ok && errResp.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	return setVPCState(d, vpc)
}

func resourceVPCUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)

	req := &raff.UpdateVPCRequest{}
	if d.HasChange("name") {
		req.Name = raff.String(d.Get("name").(string))
	}
	if d.HasChange("description") {
		req.Description = raff.String(d.Get("description").(string))
	}

	vpc, _, err := client.VPCs.Update(ctx, d.Id(), req)
	if err != nil {
		return diag.FromErr(err)
	}

	return setVPCState(d, vpc)
}

func resourceVPCDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)

	_, err := client.VPCs.Delete(ctx, d.Id())
	if err != nil {
		if errResp, ok := err.(*raff.ErrorResponse); ok && errResp.StatusCode == 404 {
			return nil
		}
		return diag.FromErr(err)
	}

	return nil
}

func setVPCState(d *schema.ResourceData, v *raff.VPC) diag.Diagnostics {
	d.Set("name", v.Name)
	d.Set("cidr", v.Cidr)
	d.Set("region", string(v.Region))
	d.Set("status", v.Status)
	d.Set("dns", raff.StringValue(v.DNS))
	d.Set("gateway", raff.StringValue(v.Gateway))
	d.Set("gateway_type", raff.StringValue(v.GatewayType))
	d.Set("router_public_ip", raff.StringValue(v.RouterPublicIP))
	d.Set("router_status", raff.StringValue(v.RouterStatus))
	if v.AccountID != nil {
		d.Set("account_id", v.AccountID.String())
	}
	if v.ProjectID != nil {
		d.Set("project_id", v.ProjectID.String())
	}
	if v.TotalIps != nil {
		d.Set("total_ips", *v.TotalIps)
	}
	if v.UsedIps != nil {
		d.Set("used_ips", *v.UsedIps)
	}
	if v.CreatedAt != nil {
		d.Set("created_at", v.CreatedAt.Format("2006-01-02T15:04:05Z"))
	}
	if v.UpdatedAt != nil {
		d.Set("updated_at", v.UpdatedAt.Format("2006-01-02T15:04:05Z"))
	}

	return nil
}
