package provider

import (
	"context"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	raff "github.com/rafftechnologies/raff-go"
	"github.com/rafftechnologies/raff-go/spec"
)

func resourceVolume() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a Raff block storage volume.",
		CreateContext: resourceVolumeCreate,
		ReadContext:   resourceVolumeRead,
		UpdateContext: resourceVolumeUpdate,
		DeleteContext: resourceVolumeDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Volume display name.",
			},
			"size": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Size in GB. Updates trigger an in-place resize (must be larger than current).",
			},
			"volume_type": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Storage class. One of: `nvme`.",
			},
			"region": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Region. Defaults to the project's default region.",
			},
			"filesystem_type": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Filesystem type for first attach (Linux only). Defaults to ext4.",
			},
			"vm_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "VM UUID to attach this volume to. Updates trigger detach/reattach.",
			},
			// Computed
			"status":         {Type: schema.TypeString, Computed: true},
			"price_per_hour": {Type: schema.TypeString, Computed: true},
			"account_id":     {Type: schema.TypeString, Computed: true},
			"project_id":     {Type: schema.TypeString, Computed: true},
			"created_at":     {Type: schema.TypeString, Computed: true},
			"updated_at":     {Type: schema.TypeString, Computed: true},
		},
	}
}

func resourceVolumeCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)

	req := &raff.CreateVolumeRequest{
		Name:       d.Get("name").(string),
		Size:       d.Get("size").(int),
		VolumeType: spec.CreateVolumeRequestVolumeType(d.Get("volume_type").(string)),
	}
	if v, ok := d.GetOk("region"); ok {
		r := spec.CreateVolumeRequestRegion(v.(string))
		req.Region = &r
	}
	if v, ok := d.GetOk("filesystem_type"); ok {
		f := spec.CreateVolumeRequestFilesystemType(v.(string))
		req.FilesystemType = &f
	}
	if v, ok := d.GetOk("vm_id"); ok {
		id, err := uuid.Parse(v.(string))
		if err != nil {
			return diag.Errorf("invalid vm_id: %s", err)
		}
		req.VMID = &id
	}

	vol, _, err := client.Volumes.Create(ctx, req)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(itoa(vol.ID))
	return setVolumeState(d, vol)
}

func resourceVolumeRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	id, err := atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	vol, _, err := client.Volumes.Get(ctx, id)
	if err != nil {
		if errResp, ok := err.(*raff.ErrorResponse); ok && errResp.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	return setVolumeState(d, vol)
}

func resourceVolumeUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	id, err := atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChange("size") {
		newSize := d.Get("size").(int)
		if _, _, err := client.Volumes.Resize(ctx, id, &raff.ResizeVolumeRequest{NewSize: newSize}); err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange("vm_id") {
		// detach old, attach new
		old, current := d.GetChange("vm_id")
		if old.(string) != "" {
			if _, err := client.Volumes.Detach(ctx, id); err != nil {
				return diag.FromErr(err)
			}
		}
		if current.(string) != "" {
			vid, err := uuid.Parse(current.(string))
			if err != nil {
				return diag.Errorf("invalid vm_id: %s", err)
			}
			if _, _, err := client.Volumes.Attach(ctx, id, &raff.AttachVolumeRequest{VMID: vid}); err != nil {
				return diag.FromErr(err)
			}
		}
	}

	return resourceVolumeRead(ctx, d, meta)
}

func resourceVolumeDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	id, err := atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if _, err := client.Volumes.Delete(ctx, id); err != nil {
		if errResp, ok := err.(*raff.ErrorResponse); ok && errResp.StatusCode == 404 {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}

func setVolumeState(d *schema.ResourceData, v *raff.Volume) diag.Diagnostics {
	d.Set("name", v.Name)
	d.Set("size", v.Size)
	d.Set("volume_type", string(v.VolumeType))
	d.Set("status", string(v.Status))
	d.Set("price_per_hour", raff.StringValue(v.PricePerHour))
	if v.Region != nil {
		d.Set("region", string(*v.Region))
	}
	if v.AccountID != nil {
		d.Set("account_id", v.AccountID.String())
	}
	if v.ProjectID != nil {
		d.Set("project_id", v.ProjectID.String())
	}
	if v.ProductVM != nil {
		d.Set("vm_id", v.ProductVM.String())
	} else {
		d.Set("vm_id", "")
	}
	if v.CreatedAt != nil {
		d.Set("created_at", v.CreatedAt.Format("2006-01-02T15:04:05Z"))
	}
	if v.UpdatedAt != nil {
		d.Set("updated_at", v.UpdatedAt.Format("2006-01-02T15:04:05Z"))
	}
	return nil
}
