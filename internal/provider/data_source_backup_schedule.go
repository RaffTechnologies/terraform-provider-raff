package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	raff "github.com/rafftechnologies/raff-go"
)

func backupScheduleComputedSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"id":             {Type: schema.TypeInt, Computed: true},
		"name":           {Type: schema.TypeString, Computed: true},
		"vm_id":          {Type: schema.TypeString, Computed: true},
		"frequency":      {Type: schema.TypeString, Computed: true},
		"keep_count":     {Type: schema.TypeInt, Computed: true},
		"runtime":        {Type: schema.TypeString, Computed: true},
		"region":         {Type: schema.TypeString, Computed: true},
		"price_per_hour": {Type: schema.TypeString, Computed: true},
	}
}

func dataSourceBackupSchedule() *schema.Resource {
	s := backupScheduleComputedSchema()
	s["id"] = &schema.Schema{Type: schema.TypeInt, Required: true}
	return &schema.Resource{
		Description: "Reads a single backup schedule by ID.",
		ReadContext: dataSourceBackupScheduleRead,
		Schema:      s,
	}
}

func dataSourceBackupScheduleRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	s, _, err := client.BackupSchedules.Get(ctx, d.Get("id").(int))
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(itoa(s.ID))
	for k, v := range backupScheduleToMap(s) {
		d.Set(k, v)
	}
	return nil
}

func dataSourceBackupSchedules() *schema.Resource {
	return &schema.Resource{
		Description: "Lists backup schedules in the current project.",
		ReadContext: dataSourceBackupSchedulesRead,
		Schema: map[string]*schema.Schema{
			"backup_schedules": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Resource{Schema: backupScheduleComputedSchema()},
			},
		},
	}
}

func dataSourceBackupSchedulesRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	items, _, err := client.BackupSchedules.List(ctx, nil)
	if err != nil {
		return diag.FromErr(err)
	}
	out := make([]map[string]any, 0, len(items))
	for i := range items {
		out = append(out, backupScheduleToMap(&items[i]))
	}
	d.SetId("backup_schedules")
	d.Set("backup_schedules", out)
	return nil
}

func backupScheduleToMap(s *raff.BackupSchedule) map[string]any {
	m := map[string]any{
		"id":             s.ID,
		"name":           s.Name,
		"frequency":      string(s.Type),
		"keep_count":     s.KeepCount,
		"price_per_hour": raff.StringValue(s.PricePerHour),
	}
	if s.ProductVM != nil {
		m["vm_id"] = s.ProductVM.String()
	}
	if s.Runtime != nil {
		m["runtime"] = *s.Runtime
	}
	if s.Region != nil {
		m["region"] = string(*s.Region)
	}
	return m
}
