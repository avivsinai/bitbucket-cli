package discovery

import (
	"reflect"
	"testing"
)

func TestFindNameCollisions(t *testing.T) {
	skills := []Skill{
		{Name: "review", Convention: "skills"},
		{Name: "review", Namespace: "acme", Convention: "skills-namespaced"},
		{Name: "unique", Convention: "skills"},
		{Name: "deploy", Namespace: "bot", Convention: "plugins"},
		{Name: "deploy", Convention: "root"},
	}
	got := FindNameCollisions(skills)
	want := []NameCollision{
		{Name: "deploy", DisplayNames: []string{"[plugins] bot/deploy", "[root] deploy"}},
		{Name: "review", DisplayNames: []string{"review", "acme/review"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("collisions = %+v, want %+v", got, want)
	}

	formatted := FormatCollisions(got)
	if formatted != "deploy: [plugins] bot/deploy, [root] deploy\n  review: review, acme/review" {
		t.Fatalf("FormatCollisions = %q", formatted)
	}

	if c := FindNameCollisions(skills[2:4]); c != nil {
		t.Fatalf("expected no collisions, got %+v", c)
	}
}
