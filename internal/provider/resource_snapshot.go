package provider

import (
	"context"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	raff "github.com/rafftechnologies/raff-go"
	"github.com/rafftechnologies/raff-go/spec"
)

func resourceSnapshot() *schema.Resource {
	return &schema.Resource{
		Description:   "Captures a point-in-time snapshot of a VM disk or a volume. Mirrors the digitalocean_droplet_snapshot / digitalocean_volume_snapshot pattern — once taken, Terraform owns the snapshot lifecycle (rename / delete).",
		CreateContext: resourceSnapshotCreate,
		ReadContext:   resourceSnapshotRead,
		UpdateContext: resourceSnapshotUpdate,
		DeleteContext: resourceSnapshotDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Snapshot display name. Updates trigger a rename.",
			},
			"resource_type": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Source type: vm or volume.",
			},
			"vm_id": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Source VM UUID. Required when resource_type=vm.",
			},
			"volume_id": {
				Type:        schema.TypeInt,
				Optional:    true,
				ForceNew:    true,
				Description: "Source volume ID. Required when resource_type=volume.",
			},
			// Computed
			"size":       {Type: schema.TypeString, Computed: true},
			"status":     {Type: schema.TypeString, Computed: true},
			"created_at": {Type: schema.TypeString, Computed: true},
		},
	}
}

func resourceSnapshotCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)

	req := &raff.CreateSnapshotRequest{
		Name:         d.Get("name").(string),
		ResourceType: spec.CreateSnapshotRequestResourceType(d.Get("resource_type").(string)),
	}
	if v, ok := d.GetOk("vm_id"); ok {
		id, err := uuid.Parse(v.(string))
		if err != nil {
			return diag.Errorf("invalid vm_id: %s", err)
		}
		req.ResourceID = &id
	}
	if v, ok := d.GetOk("volume_id"); ok {
		vid := v.(int)
		req.VolumeID = &vid
	}

	s, _, err := client.Snapshots.Create(ctx, req)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(itoa(s.ID))
	return setSnapshotState(d, s)
}

func resourceSnapshotRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	id, err := atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	s, _, err := client.Snapshots.Get(ctx, id)
	if err != nil {
		if errResp, ok := err.(*raff.ErrorResponse); ok && errResp.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	return setSnapshotState(d, s)
}

func resourceSnapshotUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	id, err := atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if d.HasChange("name") {
		s, _, err := client.Snapshots.Rename(ctx, id, &raff.RenameSnapshotRequest{Name: d.Get("name").(string)})
		if err != nil {
			return diag.FromErr(err)
		}
		return setSnapshotState(d, s)
	}
	return resourceSnapshotRead(ctx, d, meta)
}

func resourceSnapshotDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	id, err := atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if _, err := client.Snapshots.Delete(ctx, id); err != nil {
		if errResp, ok := err.(*raff.ErrorResponse); ok && errResp.StatusCode == 404 {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}

func setSnapshotState(d *schema.ResourceData, s *raff.Snapshot) diag.Diagnostics {
	d.Set("name", s.Name)
	d.Set("resource_type", string(s.Type))
	if s.ProductVM != nil {
		d.Set("vm_id", s.ProductVM.String())
	}
	if s.Size != nil {
		d.Set("size", *s.Size)
	}
	if s.Status != nil {
		d.Set("status", string(*s.Status))
	}
	if s.CreatedAt != nil {
		d.Set("created_at", s.CreatedAt.Format("2006-01-02T15:04:05Z"))
	}
	return nil
}
