package catalog

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadUsesAntoraMetadataAndPagesDirectory(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "component")
	pages := filepath.Join(source, "modules", "ROOT", "pages")
	if err := os.MkdirAll(pages, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "antora.yml"), []byte("name: api\ntitle: API Guide\nversion: 2.0\nnav:\n  - modules/ROOT/nav.adoc\n"), 0644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "dockube.yml")
	if err := os.WriteFile(path, []byte("site:\n  title: Team Docs\ncontent:\n  sources:\n    - url: ./component\n"), 0644); err != nil {
		t.Fatal(err)
	}
	c, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	s := c.Content.Sources[0]
	if s.Component != "api" || s.Title != "API Guide" || s.Version != "2.0" || s.StartPath != pages {
		t.Fatalf("source = %#v", s)
	}
}
