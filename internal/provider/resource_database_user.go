package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	raff "github.com/rafftechnologies/raff-go"
)

func resourceDatabaseUser() *schema.Resource {
	return &schema.Resource{
		Description: "Manages a user inside a managed database, with read-only or read-write access.",

		CreateContext: resourceDatabaseUserCreate,
		ReadContext:   resourceDatabaseUserRead,
		DeleteContext: resourceDatabaseUserDelete,

		Importer: &schema.ResourceImporter{
			StateContext: resourceDatabaseUserImport,
		},

		Schema: map[string]*schema.Schema{
			"database_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Short ID of the database (`raff_database.<name>.database_id`).",
			},
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateDatabaseUserName,
				Description:  "3 to 31 characters: a lowercase letter, then lowercase letters, digits or underscores. Names starting with `app_` or `fnb_` are reserved for Raff Apps and Functions.",
			},
			"role": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "readonly",
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{"readonly", "readwrite"}, false),
				Description:  "Access: `readonly` (default) or `readwrite`.",
			},
			"password": {Type: schema.TypeString, Computed: true, Sensitive: true},
			"scope":    {Type: schema.TypeString, Computed: true, Description: "The database, schema or keyspace the access applies to."},
		},
	}
}

func resourceDatabaseUserCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	dbID := d.Get("database_id").(string)
	name := d.Get("name").(string)
	_, password, _, err := client.Databases.CreateUser(ctx, dbID, name, raff.DatabaseUserRole(d.Get("role").(string)))
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(dbID + "/" + name)
	d.Set("password", password)
	return resourceDatabaseUserRead(ctx, d, meta)
}

func resourceDatabaseUserRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	dbID, name, err := splitDatabaseUserID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	users, _, err := client.Databases.ListUsers(ctx, dbID)
	if err != nil {
		if errResp, ok := err.(*raff.ErrorResponse); ok && errResp.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	var found *raff.DatabaseUser
	for i := range users {
		if raff.StringValue(users[i].Name) == name {
			found = &users[i]
			break
		}
	}
	if found == nil {
		d.SetId("")
		return nil
	}
	d.Set("database_id", dbID)
	d.Set("name", name)
	d.Set("scope", raff.StringValue(found.Scope))
	d.Set("role", databaseUserRole(raff.StringValue(found.Role)))
	if cred, _, err := client.Databases.UserCredential(ctx, dbID, name); err == nil {
		d.Set("password", cred.Password)
	}
	return nil
}

func resourceDatabaseUserDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*raff.Client)
	dbID, name, err := splitDatabaseUserID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if _, err := client.Databases.DeleteUser(ctx, dbID, name); err != nil {
		if errResp, ok := err.(*raff.ErrorResponse); ok && errResp.StatusCode == 404 {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}

// resourceDatabaseUserImport accepts `<database_id>/<name>`.
func resourceDatabaseUserImport(ctx context.Context, d *schema.ResourceData, meta any) ([]*schema.ResourceData, error) {
	if _, _, err := splitDatabaseUserID(d.Id()); err != nil {
		return nil, err
	}
	return []*schema.ResourceData{d}, nil
}

var databaseUserNameRe = regexp.MustCompile(`^[a-z][a-z0-9_]{2,30}$`)

// validateDatabaseUserName mirrors the API's rule so a bad name fails at plan.
func validateDatabaseUserName(v any, key string) ([]string, []error) {
	s, _ := v.(string)
	switch {
	case !databaseUserNameRe.MatchString(s):
		return nil, []error{fmt.Errorf("%s: %q must be 3 to 31 characters: a lowercase letter, then lowercase letters, digits or underscores", key, s)}
	case strings.HasPrefix(s, "app_") || strings.HasPrefix(s, "fnb_"):
		return nil, []error{fmt.Errorf("%s: %q is reserved: names starting with app_ or fnb_ are used by Raff Apps and Functions", key, s)}
	}
	return nil, nil
}

func splitDatabaseUserID(id string) (string, string, error) {
	dbID, name, ok := strings.Cut(id, "/")
	if !ok || dbID == "" || name == "" {
		return "", "", fmt.Errorf("invalid database user ID %q: expected <database_id>/<name>", id)
	}
	return dbID, name, nil
}

// databaseUserRole maps the engine's access summary ("read only",
// "read/write") back to the role it was created with.
func databaseUserRole(access string) string {
	if strings.Contains(access, "write") {
		return "readwrite"
	}
	return "readonly"
}
