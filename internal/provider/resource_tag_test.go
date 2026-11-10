package provider

import (
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	raff "github.com/rafftechnologies/raff-go"
)

func TestResourceTagSchema(t *testing.T) {
	if err := resourceTag().InternalValidate(nil, true); err != nil {
		t.Fatal(err)
	}
	if err := resourceSecurityGroup().InternalValidate(nil, true); err != nil {
		t.Fatal(err)
	}
}

// A tag without a priority reads back as 0, the value of a configuration that leaves priority out, so it plans no
// change; a tag with no colour reads back as the default.
func TestSetTagState(t *testing.T) {
	d := resourceTag().TestResourceData()
	two := 2
	setTagState(d, &raff.Tag{Name: "web", Color: "blue", Priority: &two, ServerCount: 3})
	if d.Get("name") != "web" || d.Get("color") != "blue" || d.Get("priority") != 2 || d.Get("server_count") != 3 {
		t.Errorf("with priority: %v %v %v %v", d.Get("name"), d.Get("color"), d.Get("priority"), d.Get("server_count"))
	}
	setTagState(d, &raff.Tag{Name: "web"})
	if d.Get("color") != "gray" || d.Get("priority") != 0 {
		t.Errorf("without priority: %v %v", d.Get("color"), d.Get("priority"))
	}
}

func TestFindTag(t *testing.T) {
	a, b := uuid.New(), uuid.New()
	tags := []raff.Tag{{ID: a, Name: "a"}, {ID: b, Name: "b"}}
	if tag := findTag(tags, b.String()); tag == nil || tag.Name != "b" {
		t.Errorf("found %+v", tag)
	}
	if tag := findTag(tags, uuid.New().String()); tag != nil {
		t.Errorf("found %+v for a missing tag", tag)
	}
}

func TestExpandTagIDs(t *testing.T) {
	set := schema.NewSet(schema.HashString, []any{"b", "a"})
	if got := expandTagIDs(set); len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("got %v", got)
	}
	if got := expandTagIDs(schema.NewSet(schema.HashString, nil)); got == nil || len(got) != 0 {
		t.Errorf("empty set: %v", got)
	}
}
