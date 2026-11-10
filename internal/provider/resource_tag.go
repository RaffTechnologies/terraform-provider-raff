package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	raff "github.com/rafftechnologies/raff-go"
)

// tagColors are the colours a tag can have.
var tagColors = []string{"gray", "red", "orange", "amber", "green", "blue", "violet", "pink"}

func resourceTag() *schema.Resource {
	return &schema.Resource{
		Description: "Manages an account tag. A tag is shared by every server that has it: a change shows on all of them. " +
			"Link a tag to a firewall with `tag_ids` on `raff_security_group`. Needs the `account.tags.manage` permission " +
			"(account Owner and Admin by default).",
		CreateContext: resourceTagCreate,
		ReadContext:   resourceTagRead,
		UpdateContext: resourceTagUpdate,
		DeleteContext: resourceTagDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringLenBetween(1, 40),
				Description:  "Tag name, 1-40 characters, unique in the account ignoring case.",
			},
			"color": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "gray",
				ValidateFunc: validation.StringInSlice(tagColors, false),
				Description:  "Tag colour: `gray` (default), `red`, `orange`, `amber`, `green`, `blue`, `violet` or `pink`.",
			},
			"priority": {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntBetween(1, 9),
				Description:  "Priority 1-9, 1 highest. Leave it out for none.",
			},
			// Computed
			"server_count": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Servers with this tag, in the projects the API key can see.",
			},
			"created_at": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"updated_at": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceTagCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)

	color := raff.TagColor(d.Get("color").(string))
	req := &raff.CreateTagRequest{Name: d.Get("name").(string), Color: &color}
	if v, ok := d.GetOk("priority"); ok {
		req.Priority = raff.Int(v.(int))
	}
	tag, _, err := client.Tags.Create(ctx, req)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(tag.ID.String())
	return setTagState(d, tag)
}

// resourceTagRead finds the tag in the account's list: the API has no single-tag read.
func resourceTagRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)

	tags, _, err := client.Tags.List(ctx, nil)
	if err != nil {
		return diag.FromErr(err)
	}
	tag := findTag(tags, d.Id())
	if tag == nil {
		d.SetId("")
		return nil
	}
	return setTagState(d, tag)
}

func resourceTagUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)

	req := &raff.UpdateTagRequest{}
	if d.HasChange("name") {
		req.Name = raff.String(d.Get("name").(string))
	}
	if d.HasChange("color") {
		color := raff.TagColor(d.Get("color").(string))
		req.Color = &color
	}
	if d.HasChange("priority") {
		// 0 (priority left out) removes it.
		req.Priority = raff.Int(d.Get("priority").(int))
	}
	tag, _, err := client.Tags.Update(ctx, d.Id(), req)
	if err != nil {
		return diag.FromErr(err)
	}
	return setTagState(d, tag)
}

func resourceTagDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	if _, err := client.Tags.Delete(ctx, d.Id()); err != nil {
		if errResp, ok := err.(*raff.ErrorResponse); ok && errResp.StatusCode == 404 {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}

func findTag(tags []raff.Tag, id string) *raff.Tag {
	for i := range tags {
		if tags[i].ID.String() == id {
			return &tags[i]
		}
	}
	return nil
}

func setTagState(d *schema.ResourceData, tag *raff.Tag) diag.Diagnostics {
	d.Set("name", tag.Name)
	color := string(tag.Color)
	if color == "" {
		color = "gray"
	}
	d.Set("color", color)
	priority := 0
	if tag.Priority != nil {
		priority = *tag.Priority
	}
	d.Set("priority", priority)
	d.Set("server_count", tag.ServerCount)
	d.Set("created_at", tag.CreatedAt.Format("2006-01-02T15:04:05Z"))
	d.Set("updated_at", tag.UpdatedAt.Format("2006-01-02T15:04:05Z"))
	return nil
}
