package provider

import (
	"testing"

	raff "github.com/rafftechnologies/raff-go"
	"github.com/rafftechnologies/raff-go/spec"
)

// The firewall goes on the primary public IPv4 interface; without one marked primary, on the first public IPv4; a VM
// with no public IPv4 has none.
func TestPrimaryPublicNIC(t *testing.T) {
	yes, no := true, false
	nets := []raff.VMNetwork{
		{NicID: 0, Type: spec.VMNetworkTypeVpc},
		{NicID: 1, Type: spec.VMNetworkTypePublic, IsPrimary: &no},
		{NicID: 2, Type: spec.VMNetworkTypePublic, IsPrimary: &yes},
		{NicID: 3, Type: spec.VMNetworkTypeIpv6},
	}
	if nic := primaryPublicNIC(nets); nic == nil || nic.NicID != 2 {
		t.Errorf("primary: %+v", nic)
	}
	if nic := primaryPublicNIC(nets[:2]); nic == nil || nic.NicID != 1 {
		t.Errorf("first public: %+v", nic)
	}
	if nic := primaryPublicNIC([]raff.VMNetwork{nets[0], nets[3]}); nic != nil {
		t.Errorf("no public IPv4: %+v", nic)
	}
}
