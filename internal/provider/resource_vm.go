package provider

import (
	"context"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	raff "github.com/rafftechnologies/raff-go"
	"github.com/rafftechnologies/raff-go/spec"
)

func resourceVM() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a Raff virtual machine.",
		CreateContext: resourceVMCreate,
		ReadContext:   resourceVMRead,
		UpdateContext: resourceVMUpdate,
		DeleteContext: resourceVMDelete,

		Schema: map[string]*schema.Schema{
			// Required input
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "VM name. Changing this renames the VM.",
			},
			"template_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "OS template ID.",
			},
			"pricing_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Pricing plan ID. Changing this resizes the VM.",
			},
			"region": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Data center region.",
			},
			// Optional input
			"password": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				ForceNew:    true,
				Description: "Root password.",
			},
			"ssh_keys": {
				Type:        schema.TypeList,
				Optional:    true,
				ForceNew:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "SSH key IDs.",
			},
			"extra_storage": {
				Type:        schema.TypeInt,
				Optional:    true,
				ForceNew:    true,
				Description: "Extra block storage in GB.",
			},
			"extra_storage_type": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Extra storage filesystem type (ext4, xfs, btrfs).",
			},
			"backup_type": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Backup type: none, daily, weekly.",
			},
			"backup_time": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Backup time (e.g. 8am).",
			},
			"backup_date": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Backup day (e.g. Saturday).",
			},
			"tags": {
				Type:        schema.TypeList,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Tag names. Updatable: tags are diffed against current state — additions are added, removals are removed.",
			},
			"vpc_id": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "VPC ID to attach.",
			},
			"vpc_name": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "VPC name (creates new VPC).",
			},
			"vpc_cidr": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "VPC CIDR block.",
			},
			"volume_action": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "detach",
				Description: "Volume action on delete: detach or delete.",
			},
			"delete_vpc": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Whether to delete VPC on VM deletion.",
			},
			// Computed (read-only)
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "VM status.",
			},
			"cpu": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Number of vCPUs.",
			},
			"ram": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "RAM in GB.",
			},
			"storage": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Base storage in GB.",
			},
			"added_storage": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Added storage in GB.",
			},
			"total_storage": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Total storage in GB.",
			},
			"template_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "OS template name.",
			},
			"template_version": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "OS template version.",
			},
			"price_per_hour": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Hourly price.",
			},
			"public_ipv4_address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Public IPv4 address.",
			},
			"private_ipv4_address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Private IPv4 address.",
			},
			"public_ipv6_address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Public IPv6 address.",
			},
			"private_ipv6_address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Private IPv6 address.",
			},
			"billing_type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Billing type.",
			},
			"active": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the VM is active.",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Creation timestamp.",
			},
			"updated_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Last update timestamp.",
			},
		},
	}
}

func resourceVMCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)

	tmplID, err := uuid.Parse(d.Get("template_id").(string))
	if err != nil {
		return diag.Errorf("invalid template_id: %s", err)
	}

	req := &raff.CreateVMRequest{
		Name:       d.Get("name").(string),
		TemplateID: tmplID,
		PricingID:  d.Get("pricing_id").(int),
		Region:     spec.CreateVMRequestRegion(d.Get("region").(string)),
	}

	if v, ok := d.GetOk("password"); ok {
		req.Password = raff.String(v.(string))
	}
	if v, ok := d.GetOk("ssh_keys"); ok {
		keys := make([]string, 0)
		for _, k := range v.([]interface{}) {
			keys = append(keys, k.(string))
		}
		req.SSHKeys = &keys
	}
	if v, ok := d.GetOk("extra_storage"); ok {
		req.ExtraStorage = raff.Int(v.(int))
	}
	if v, ok := d.GetOk("extra_storage_type"); ok {
		st := spec.CreateVMRequestExtraStorageType(v.(string))
		req.ExtraStorageType = &st
	}
	if v, ok := d.GetOk("backup_type"); ok {
		bt := spec.CreateVMRequestBackupType(v.(string))
		req.BackupType = &bt
	}
	if v, ok := d.GetOk("backup_time"); ok {
		req.BackupTime = raff.String(v.(string))
	}
	if v, ok := d.GetOk("backup_date"); ok {
		req.BackupDate = raff.String(v.(string))
	}
	if v, ok := d.GetOk("tags"); ok {
		tags := make([]string, 0)
		for _, t := range v.([]interface{}) {
			tags = append(tags, t.(string))
		}
		req.Tags = &tags
	}
	if v, ok := d.GetOk("vpc_id"); ok {
		id, err := uuid.Parse(v.(string))
		if err != nil {
			return diag.Errorf("invalid vpc_id: %s", err)
		}
		req.VpcID = &id
	}
	if v, ok := d.GetOk("vpc_name"); ok {
		req.VpcName = raff.String(v.(string))
	}
	if v, ok := d.GetOk("vpc_cidr"); ok {
		req.VpcCidr = raff.String(v.(string))
	}

	vm, _, err := client.VMs.Create(ctx, req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(vm.ID.String())

	return setVMState(d, vm)
}

func resourceVMRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)

	vm, _, err := client.VMs.Get(ctx, d.Id())
	if err != nil {
		if errResp, ok := err.(*raff.ErrorResponse); ok && errResp.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	return setVMState(d, vm)
}

func resourceVMUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)

	if d.HasChange("name") {
		newName := d.Get("name").(string)
		_, err := client.VMs.Rename(ctx, d.Id(), &raff.RenameVMRequest{Name: newName})
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange("pricing_id") {
		newPricingID := d.Get("pricing_id").(int)
		_, _, err := client.VMs.Resize(ctx, d.Id(), &raff.ResizeVMRequest{PricingID: newPricingID})
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange("tags") {
		if err := syncVMTags(ctx, client, d); err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceVMRead(ctx, d, meta)
}

// syncVMTags diffs the desired tag-name set against the VM's current tags and
// applies AddTag / RemoveTag calls so the VM ends up with exactly the desired set.
func syncVMTags(ctx context.Context, client *raff.Client, d *schema.ResourceData) error {
	desired := map[string]struct{}{}
	for _, t := range d.Get("tags").([]any) {
		desired[t.(string)] = struct{}{}
	}

	vm, _, err := client.VMs.Get(ctx, d.Id())
	if err != nil {
		return err
	}
	current := map[string]string{} // name -> tag ID
	if vm.Tags != nil {
		for _, t := range *vm.Tags {
			current[t.Name] = t.ID
		}
	}

	for name := range desired {
		if _, ok := current[name]; ok {
			continue
		}
		if _, _, err := client.VMs.AddTag(ctx, d.Id(), &raff.AddVMTagRequest{Name: name}); err != nil {
			return err
		}
	}
	for name, id := range current {
		if _, ok := desired[name]; ok {
			continue
		}
		if _, _, err := client.VMs.RemoveTag(ctx, d.Id(), id); err != nil {
			return err
		}
	}
	return nil
}

func resourceVMDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)

	req := &raff.DeleteVMRequest{}
	if v, ok := d.GetOk("volume_action"); ok {
		va := spec.DeleteVMRequestVolumeAction(v.(string))
		req.VolumeAction = &va
	}
	if v, ok := d.GetOk("delete_vpc"); ok {
		req.DeleteVpc = raff.Bool(v.(bool))
	}

	_, err := client.VMs.Delete(ctx, d.Id(), req)
	if err != nil {
		if errResp, ok := err.(*raff.ErrorResponse); ok && errResp.StatusCode == 404 {
			return nil
		}
		return diag.FromErr(err)
	}

	return nil
}

func setVMState(d *schema.ResourceData, vm *raff.VM) diag.Diagnostics {
	d.Set("name", vm.Name)
	d.Set("status", string(vm.Status))
	d.Set("cpu", vm.CPU)
	d.Set("ram", vm.RAM)
	d.Set("storage", vm.Storage)
	d.Set("added_storage", vm.AddedStorage)
	d.Set("total_storage", vm.TotalStorage)
	d.Set("template_id", vm.TemplateID.String())
	d.Set("template_name", vm.TemplateName)
	d.Set("template_version", vm.TemplateVersion)
	d.Set("pricing_id", vm.PricingID)
	d.Set("price_per_hour", vm.PricePerHour)
	d.Set("region", string(vm.Region))
	d.Set("public_ipv4_address", raff.StringValue(vm.PublicIpv4Address))
	d.Set("private_ipv4_address", raff.StringValue(vm.PrivateIpv4Address))
	d.Set("public_ipv6_address", raff.StringValue(vm.PublicIpv6Address))
	d.Set("private_ipv6_address", raff.StringValue(vm.PrivateIpv6Address))
	if vm.BillingType != nil {
		d.Set("billing_type", string(*vm.BillingType))
	} else {
		d.Set("billing_type", "")
	}
	d.Set("active", vm.Active)
	d.Set("created_at", vm.CreatedAt.Format("2006-01-02T15:04:05Z"))
	d.Set("updated_at", vm.UpdatedAt.Format("2006-01-02T15:04:05Z"))

	tagNames := []string{}
	if vm.Tags != nil {
		for _, t := range *vm.Tags {
			tagNames = append(tagNames, t.Name)
		}
	}
	d.Set("tags", tagNames)

	return nil
}
