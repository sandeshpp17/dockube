package markdown

import (
	"strings"
	"testing"
)

func TestRenderFeatures(t *testing.T) {
	r, err := Render("---\ntitle: Guide\ntags: go, docs\nowner: platform\n---\n# ignored\n\n| A | B |\n| - | - |\n| 1 | 2 |\n\n```go\nfmt.Println(\"ok\")\n```\n\n[[install|Install]]", "api", "1.0")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Guide", "color:", "/docs/api/1.0/install"} {
		if !strings.Contains(r.HTML, want) && r.Title != want {
			t.Errorf("rendered HTML missing %q: %s", want, r.HTML)
		}
	}
	if len(r.Tags) != 2 || r.Owner != "platform" {
		t.Fatalf("metadata = %#v, owner=%q", r.Tags, r.Owner)
	}
}
