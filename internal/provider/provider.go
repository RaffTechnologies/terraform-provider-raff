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
			// Existing
			"raff_project":        resourceProject(),
			"raff_vm":             resourceVM(),
			"raff_vpc":            resourceVPC(),
			"raff_ip":             resourceIP(),
			"raff_security_group": resourceSecurityGroup(),
			// Wave 1 — VM-adjacent storage
			"raff_ssh_key":         resourceSSHKey(),
			"raff_volume":          resourceVolume(),
			"raff_snapshot":        resourceSnapshot(),
			"raff_backup":          resourceBackup(),
			"raff_backup_schedule": resourceBackupSchedule(),
			// Wave 2 — Identity / RBAC
			"raff_api_key":        resourceAPIKey(),
			"raff_role":           resourceRole(),
			"raff_member":         resourceMember(),
			"raff_project_member": resourceProjectMember(),
		},

		DataSourcesMap: map[string]*schema.Resource{
			// Existing
			"raff_project":         dataSourceProject(),
			"raff_projects":        dataSourceProjects(),
			"raff_vm":              dataSourceVM(),
			"raff_vms":             dataSourceVMs(),
			"raff_vm_networks":     dataSourceVMNetworks(),
			"raff_vpc":             dataSourceVPC(),
			"raff_vpcs":            dataSourceVPCs(),
			"raff_vpc_cidr_suggestions": dataSourceVPCCIDRSuggestions(),
			"raff_ip":              dataSourceIP(),
			"raff_ips":             dataSourceIPs(),
			"raff_security_group":  dataSourceSecurityGroup(),
			"raff_security_groups": dataSourceSecurityGroups(),
			"raff_security_group_templates": dataSourceSecurityGroupTemplates(),
			// Wave 1
			"raff_ssh_key":          dataSourceSSHKey(),
			"raff_ssh_keys":         dataSourceSSHKeys(),
			"raff_volume":           dataSourceVolume(),
			"raff_volumes":          dataSourceVolumes(),
			"raff_snapshot":         dataSourceSnapshot(),
			"raff_snapshots":        dataSourceSnapshots(),
			"raff_backup":           dataSourceBackup(),
			"raff_backups":          dataSourceBackups(),
			"raff_backup_schedule":  dataSourceBackupSchedule(),
			"raff_backup_schedules": dataSourceBackupSchedules(),
			// Wave 2
			"raff_api_key":         dataSourceAPIKey(),
			"raff_api_keys":        dataSourceAPIKeys(),
			"raff_role":            dataSourceRole(),
			"raff_roles":           dataSourceRoles(),
			"raff_member":          dataSourceMember(),
			"raff_members":         dataSourceMembers(),
			"raff_project_members": dataSourceProjectMembers(),
			// Read-only catalog (no resource counterpart)
			"raff_regions":          dataSourceRegions(),
			"raff_templates":        dataSourceTemplates(),
			"raff_permissions":      dataSourcePermissions(),
			"raff_vm_pricing":       dataSourceVMPricing(),
			"raff_volume_pricing":   dataSourceStoragePricing("volume"),
			"raff_backup_pricing":   dataSourceStoragePricing("backup"),
			"raff_snapshot_pricing": dataSourceStoragePricing("snapshot"),
			"raff_ip_pricing":       dataSourceIPPricing(),
		},

		ConfigureContextFunc: configure,
	}
}

func configure(ctx context.Context, d *schema.ResourceData) (any, diag.Diagnostics) {
	apiKey := d.Get("api_key").(string)

	opts := []raff.ClientOpt{
		raff.SetUserAgent("terraform-provider-raff/0.1.1"),
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
