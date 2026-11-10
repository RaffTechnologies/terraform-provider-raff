package provider

import (
	"context"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	raff "github.com/rafftechnologies/raff-go"
)

// readSecurityGroupTags sets tag_ids from the tags the firewall is linked to in its project. It reads them only while
// the configuration sets tag_ids (state or config holds an ID): links made outside Terraform on a firewall whose
// configuration does not set tag_ids are left alone, so upgrading the provider never plans an unlink.
func readSecurityGroupTags(ctx context.Context, client *raff.Client, d *schema.ResourceData) error {
	if d.Get("tag_ids").(*schema.Set).Len() == 0 {
		return nil
	}
	ft, _, err := client.SecurityGroups.GetTags(ctx, d.Id())
	if err != nil {
		return fmt.Errorf("reading the firewall's tags: %w", err)
	}
	ids := make([]string, 0, len(ft.Links))
	for _, l := range ft.Links {
		ids = append(ids, l.TagID.String())
	}
	d.Set("tag_ids", ids)
	return nil
}

// syncSecurityGroupTags makes tag_ids the firewall's tags in its project, in place. Raff then puts the firewall on
// every server of that project with one of the tags; an empty set unlinks every tag (servers keep the firewall).
func syncSecurityGroupTags(ctx context.Context, client *raff.Client, d *schema.ResourceData) error {
	if _, _, err := client.SecurityGroups.SetTags(ctx, d.Id(), expandTagIDs(d.Get("tag_ids").(*schema.Set))); err != nil {
		return fmt.Errorf("linking the firewall to its tags: %w", err)
	}
	return nil
}

// expandTagIDs is the set's tag IDs, sorted so requests are stable.
func expandTagIDs(set *schema.Set) []string {
	ids := make([]string, 0, set.Len())
	for _, v := range set.List() {
		ids = append(ids, v.(string))
	}
	sort.Strings(ids)
	return ids
}
