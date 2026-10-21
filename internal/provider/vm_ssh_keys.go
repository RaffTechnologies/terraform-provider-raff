package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	raff "github.com/rafftechnologies/raff-go"
	"golang.org/x/crypto/ssh"
)

// syncVMSSHKeys makes a running Linux server's root keys follow a change of ssh_keys, without recreating it: keys new
// to the list are added first, then keys gone from it are removed, so a swapped key never leaves root without one.
// Keys are compared by fingerprint (a saved key's ID and the key itself are the same key). Keys added on the server
// itself are not Terraform's: they stay, and Terraform does not read them back.
func syncVMSSHKeys(ctx context.Context, client *raff.Client, d *schema.ResourceData) error {
	oldRaw, newRaw := d.GetChange("ssh_keys")
	oldEntries, newEntries := stringList(oldRaw), stringList(newRaw)
	oldFPs, err := sshKeyFingerprints(ctx, client, oldEntries)
	if err != nil {
		return err
	}
	newFPs, err := sshKeyFingerprints(ctx, client, newEntries)
	if err != nil {
		return err
	}
	add, remove := sshKeyChanges(oldFPs, newEntries, newFPs)
	for _, entry := range add {
		if _, _, err := client.VMs.AddSSHKey(ctx, d.Id(), &raff.AddServerSSHKeyRequest{SSHKey: entry}); err != nil {
			return fmt.Errorf("adding an SSH key to the VM: %w", err)
		}
	}
	for _, fp := range remove {
		// The last key goes only when the configuration lists none.
		_, _, err := client.VMs.RemoveSSHKey(ctx, d.Id(), fp, &raff.RemoveServerSSHKeyOptions{ConfirmLast: len(newEntries) == 0})
		var apiErr *raff.ErrorResponse
		if errors.As(err, &apiErr) && apiErr.Reason == raff.SSHKeysNotOnServer {
			continue
		}
		if err != nil {
			return fmt.Errorf("removing SSH key %s from the VM: %w", fp, err)
		}
	}
	return nil
}

// sshKeyChanges returns the new entries whose key the old list did not have, and the fingerprints of the old keys the
// new list does not have.
func sshKeyChanges(oldFPs []string, newEntries, newFPs []string) (add []string, remove []string) {
	had := map[string]bool{}
	for _, fp := range oldFPs {
		had[fp] = true
	}
	keep := map[string]bool{}
	for i, fp := range newFPs {
		keep[fp] = true
		if !had[fp] {
			had[fp] = true // a key listed twice is added once
			add = append(add, newEntries[i])
		}
	}
	seen := map[string]bool{}
	for _, fp := range oldFPs {
		if !keep[fp] && !seen[fp] {
			seen[fp] = true
			remove = append(remove, fp)
		}
	}
	return add, remove
}

// sshKeyFingerprints returns each entry's SHA256 fingerprint: a saved key's, or the public key's own.
func sshKeyFingerprints(ctx context.Context, client *raff.Client, entries []string) ([]string, error) {
	fps := make([]string, len(entries))
	for i, entry := range entries {
		entry = strings.TrimSpace(entry)
		if _, err := uuid.Parse(entry); err == nil {
			key, _, err := client.SSHKeys.Get(ctx, entry)
			if err != nil {
				return nil, fmt.Errorf("reading SSH key %s: %w", entry, err)
			}
			if raff.StringValue(key.Fingerprint) == "" {
				return nil, fmt.Errorf("SSH key %s (%s) is not a valid key: delete it and add it again", entry, key.Name)
			}
			fps[i] = raff.StringValue(key.Fingerprint)
			continue
		}
		pub, _, _, _, err := ssh.ParseAuthorizedKey([]byte(entry))
		if err != nil {
			return nil, fmt.Errorf("ssh_keys entry %d is neither a saved key's ID nor a public key", i+1)
		}
		fps[i] = ssh.FingerprintSHA256(pub)
	}
	return fps, nil
}

func stringList(v any) []string {
	list, _ := v.([]any)
	out := make([]string, 0, len(list))
	for _, item := range list {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}
