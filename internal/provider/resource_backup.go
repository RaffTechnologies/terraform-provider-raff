package provider

import (
	"context"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	raff "github.com/rafftechnologies/raff-go"
)

// resourceBackup manages a single on-demand backup of a VM.
//
// Same shape as raff_snapshot: capture once, Terraform owns the backup ID,
// delete when removed from config. Backups are immutable — name is set at
// create time and ForceNew on change. For recurring policies use
// raff_backup_schedule instead. (Note: backup_schedule itself manages its
// own retention, including auto-pruning of older backups it created.
// Don't manage those backups via raff_backup or you'll fight the
// retention engine.)
//
// Incremental series: backups run as an incremental series — the first
// backup on a VM is a full baseline, and subsequent backups store only
// changed blocks. The `increment_id` attribute tells you where this row
// sits in the series (null for legacy standalone backups, 0 for the
// baseline, 1+ for appended restore points).
//
// Destruction caveat: deleting a single mid-series restore point is not
// supported by the underlying hypervisor — the API rejects it. If your
// `terraform destroy` for a raff_backup fails with "incremental backup is
// part of a chain; delete the entire chain instead", remove the whole
// series via the CLI (`raff backup delete-series <id>`) or by deleting
// the parent VM, then run `terraform state rm` to detach Terraform's
// record. A future provider version may expose a series-delete action.
func resourceBackup() *schema.Resource {
	return &schema.Resource{
		Description:   "Captures an on-demand backup of a VM. The backup is async — Terraform returns once the backup record exists; poll status via the data source if you need to wait for `ready`. Backups run as an incremental series; see `increment_id` for series position and notes on destroy semantics. For recurring backups use raff_backup_schedule.",
		CreateContext: resourceBackupCreate,
		ReadContext:   resourceBackupRead,
		DeleteContext: resourceBackupDelete,

		Schema: map[string]*schema.Schema{
			"vm_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Source VM UUID. Cannot be changed — backup the new VM instead.",
			},
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Computed:    true,
				Description: "Display name. Defaults to a timestamped name set by the API.",
			},
			// Computed
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Lifecycle status: pending, creating, ready, restoring, failed.",
			},
			"storage_size": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Backup size in MB. For incremental restore points this is the per-increment delta; for legacy standalone backups it is the full image size. Display in MB up to 1024, else convert to GB.",
			},
			"increment_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Position within the incremental backup series. `null` for legacy standalone backups, `0` for the series baseline, `1+` for appended restore points. Rows that share an underlying image form one series; deleting a mid-series row requires deleting the whole series.",
			},
			"region":      {Type: schema.TypeString, Computed: true},
			"expire_date": {Type: schema.TypeString, Computed: true},
			"account_id":  {Type: schema.TypeString, Computed: true},
			"project_id":  {Type: schema.TypeString, Computed: true},
			"created_at":  {Type: schema.TypeString, Computed: true},
		},
	}
}

func resourceBackupCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	vmID, err := uuid.Parse(d.Get("vm_id").(string))
	if err != nil {
		return diag.Errorf("invalid vm_id: %s", err)
	}
	req := &raff.CreateBackupRequest{VMID: vmID}
	if v, ok := d.GetOk("name"); ok {
		s := v.(string)
		req.Name = &s
	}
	b, _, err := client.Backups.Create(ctx, req)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(b.ID.String())
	return setBackupState(d, b)
}

func resourceBackupRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	b, _, err := client.Backups.Get(ctx, d.Id())
	if err != nil {
		if errResp, ok := err.(*raff.ErrorResponse); ok && errResp.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	return setBackupState(d, b)
}

func resourceBackupDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	if _, err := client.Backups.Delete(ctx, d.Id()); err != nil {
		if errResp, ok := err.(*raff.ErrorResponse); ok && errResp.StatusCode == 404 {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}

func setBackupState(d *schema.ResourceData, b *raff.Backup) diag.Diagnostics {
	d.Set("name", b.Name)
	d.Set("status", b.Status)
	d.Set("storage_size", b.StorageSize)
	if b.IncrementID != nil {
		d.Set("increment_id", *b.IncrementID)
	}
	if b.ProductVM != nil {
		d.Set("vm_id", b.ProductVM.String())
	}
	if b.Region != nil {
		d.Set("region", string(*b.Region))
	}
	if b.AccountID != nil {
		d.Set("account_id", b.AccountID.String())
	}
	if b.ProjectID != nil {
		d.Set("project_id", b.ProjectID.String())
	}
	if b.ExpireDate != nil {
		d.Set("expire_date", b.ExpireDate.Format("2006-01-02T15:04:05Z"))
	}
	if b.CreatedAt != nil {
		d.Set("created_at", b.CreatedAt.Format("2006-01-02T15:04:05Z"))
	}
	return nil
}
