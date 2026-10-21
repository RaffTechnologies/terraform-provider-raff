package provider

import (
	"reflect"
	"testing"
)

// New keys are added (once, in the list's order), dropped keys removed; a key kept under another form (its ID instead
// of the key) is neither.
func TestSSHKeyChanges(t *testing.T) {
	add, remove := sshKeyChanges(
		[]string{"SHA256:a", "SHA256:b", "SHA256:c"},
		[]string{"id-of-b", "ssh-ed25519 D", "ssh-ed25519 D again", "ssh-ed25519 E"},
		[]string{"SHA256:b", "SHA256:d", "SHA256:d", "SHA256:e"},
	)
	if !reflect.DeepEqual(add, []string{"ssh-ed25519 D", "ssh-ed25519 E"}) {
		t.Errorf("add %q", add)
	}
	if !reflect.DeepEqual(remove, []string{"SHA256:a", "SHA256:c"}) {
		t.Errorf("remove %q", remove)
	}
	if add, remove := sshKeyChanges(nil, nil, nil); add != nil || remove != nil {
		t.Errorf("no change: %q %q", add, remove)
	}
}
