package provider

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	raff "github.com/rafftechnologies/raff-go"
	"github.com/rafftechnologies/raff-go/spec"
)

// readVMSecurityGroup sets security_group_id from the firewall on the VM's primary public IPv4 interface. It reads it
// only once the configuration sets one (state or config holds an ID): a firewall attached outside Terraform to a VM
// whose configuration does not set security_group_id is left alone, so upgrading the provider never plans a detach.
func readVMSecurityGroup(ctx context.Context, client *raff.Client, d *schema.ResourceData) error {
	if d.Get("security_group_id").(string) == "" {
		return nil
	}
	nets, _, err := client.VMs.ListNetworks(ctx, d.Id(), nil)
	if err != nil {
		return fmt.Errorf("reading the VM's networks: %w", err)
	}
	sgID := ""
	if nic := primaryPublicNIC(nets); nic != nil && nic.SecurityGroupID != nil {
		sgID = nic.SecurityGroupID.String()
	}
	d.Set("security_group_id", sgID)
	return nil
}

// syncVMSecurityGroup makes the firewall on the VM's primary public IPv4 interface follow a change of
// security_group_id, in place: a new ID is attached (it replaces the firewall the interface had), an empty one detaches
// the interface's firewall, which leaves the Raff Default Firewall.
func syncVMSecurityGroup(ctx context.Context, client *raff.Client, d *schema.ResourceData) error {
	newID := d.Get("security_group_id").(string)
	nets, _, err := client.VMs.ListNetworks(ctx, d.Id(), nil)
	if err != nil {
		return fmt.Errorf("reading the VM's networks: %w", err)
	}
	nic := primaryPublicNIC(nets)
	if nic == nil {
		if newID == "" {
			return nil
		}
		return fmt.Errorf("the VM has no public IPv4 address for firewall %s", newID)
	}
	if newID == "" {
		if nic.SecurityGroupID == nil {
			return nil
		}
		if _, err := client.VMs.DetachSecurityGroup(ctx, d.Id(), nic.SecurityGroupID.String(), nic.NicID); err != nil {
			return fmt.Errorf("detaching the firewall from the VM: %w", err)
		}
		return nil
	}
	id, err := uuid.Parse(newID)
	if err != nil {
		return fmt.Errorf("invalid security_group_id: %w", err)
	}
	if _, err := client.VMs.AttachSecurityGroup(ctx, d.Id(), &raff.AttachSecurityGroupRequest{NicID: nic.NicID, SecurityGroupID: id}); err != nil {
		return fmt.Errorf("attaching the firewall to the VM: %w", err)
	}
	return nil
}

// primaryPublicNIC is the VM's primary public IPv4 interface, else its first public IPv4 interface, else nil.
func primaryPublicNIC(nets []raff.VMNetwork) *raff.VMNetwork {
	var first *raff.VMNetwork
	for i := range nets {
		if nets[i].Type != spec.VMNetworkTypePublic {
			continue
		}
		if nets[i].IsPrimary != nil && *nets[i].IsPrimary {
			return &nets[i]
		}
		if first == nil {
			first = &nets[i]
		}
	}
	return first
}
