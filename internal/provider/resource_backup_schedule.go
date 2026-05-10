package provider

import (
	"context"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	raff "github.com/rafftechnologies/raff-go"
	"github.com/rafftechnologies/raff-go/spec"
)

func resourceBackupSchedule() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a recurring backup schedule for a VM. (One-shot backups are imperative and not exposed as a Terraform resource — use the CLI's `raff backup create` for those.)",
		CreateContext: resourceBackupScheduleCreate,
		ReadContext:   resourceBackupScheduleRead,
		UpdateContext: resourceBackupScheduleUpdate,
		DeleteContext: resourceBackupScheduleDelete,

		Schema: map[string]*schema.Schema{
			"vm_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Source VM UUID.",
			},
			"frequency": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Schedule frequency: daily or weekly.",
			},
			"time": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Time of day (e.g. 8am, 13:00).",
			},
			"day_of_week": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Day of week (Monday-Sunday). Required when frequency=weekly.",
			},
			"keep_count": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Number of backups to retain before auto-pruning.",
			},
			// Computed
			"name":           {Type: schema.TypeString, Computed: true},
			"runtime":        {Type: schema.TypeString, Computed: true},
			"region":         {Type: schema.TypeString, Computed: true},
			"price_per_hour": {Type: schema.TypeString, Computed: true},
		},
	}
}

func resourceBackupScheduleCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)

	vmID, err := uuid.Parse(d.Get("vm_id").(string))
	if err != nil {
		return diag.Errorf("invalid vm_id: %s", err)
	}

	req := &raff.CreateBackupScheduleRequest{
		VMID:      vmID,
		Type:      spec.CreateBackupScheduleRequestType(d.Get("frequency").(string)),
		Time:      d.Get("time").(string),
		KeepCount: d.Get("keep_count").(int),
	}
	if v, ok := d.GetOk("day_of_week"); ok {
		dw := spec.CreateBackupScheduleRequestDayOfWeek(v.(string))
		req.DayOfWeek = &dw
	}

	s, _, err := client.BackupSchedules.Create(ctx, req)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(itoa(s.ID))
	return setBackupScheduleState(d, s)
}

func resourceBackupScheduleRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	id, err := atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	s, _, err := client.BackupSchedules.Get(ctx, id)
	if err != nil {
		if errResp, ok := err.(*raff.ErrorResponse); ok && errResp.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	return setBackupScheduleState(d, s)
}

func resourceBackupScheduleUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	id, err := atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	req := &raff.UpdateBackupScheduleRequest{}
	if d.HasChange("frequency") {
		t := spec.UpdateBackupScheduleRequestType(d.Get("frequency").(string))
		req.Type = &t
	}
	if d.HasChange("time") {
		v := d.Get("time").(string)
		req.Time = &v
	}
	if d.HasChange("day_of_week") {
		dw := spec.UpdateBackupScheduleRequestDayOfWeek(d.Get("day_of_week").(string))
		req.DayOfWeek = &dw
	}
	if d.HasChange("keep_count") {
		v := d.Get("keep_count").(int)
		req.KeepCount = &v
	}

	s, _, err := client.BackupSchedules.Update(ctx, id, req)
	if err != nil {
		return diag.FromErr(err)
	}
	return setBackupScheduleState(d, s)
}

func resourceBackupScheduleDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	id, err := atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if _, err := client.BackupSchedules.Delete(ctx, id); err != nil {
		if errResp, ok := err.(*raff.ErrorResponse); ok && errResp.StatusCode == 404 {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}

func setBackupScheduleState(d *schema.ResourceData, s *raff.BackupSchedule) diag.Diagnostics {
	d.Set("name", s.Name)
	d.Set("frequency", string(s.Type))
	d.Set("keep_count", s.KeepCount)
	if s.Runtime != nil {
		d.Set("runtime", *s.Runtime)
	}
	if s.Region != nil {
		d.Set("region", string(*s.Region))
	}
	d.Set("price_per_hour", raff.StringValue(s.PricePerHour))
	if s.ProductVM != nil {
		d.Set("vm_id", s.ProductVM.String())
	}
	return nil
}
