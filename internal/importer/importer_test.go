package importer

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/dockube/dockube/internal/db"
	"github.com/dockube/dockube/internal/models"
)

func TestImportAndSearch(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "docs")
	if err := os.MkdirAll(filepath.Join(src, "guide"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "guide", "install.md"), []byte("---\ntitle: Install\ntags: setup\n---\n# Install\nRun dockube install."), 0644); err != nil {
		t.Fatal(err)
	}
	database, err := db.Open(context.Background(), filepath.Join(dir, "dockube.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	store := models.Store{DB: database}
	n, err := (Importer{Store: store}).Run(context.Background(), Job{SourceDir: src, Product: "dockube", Version: "1.0"})
	if err != nil || n != 1 {
		t.Fatalf("import = %d, %v", n, err)
	}
	v, err := store.ResolveVersion(context.Background(), "dockube", "latest")
	if err != nil {
		t.Fatal(err)
	}
	d, err := store.Document(context.Background(), v.ID, "guide/install")
	if err != nil || d.Title != "Install" {
		t.Fatalf("document=%#v err=%v", d, err)
	}
	results, err := store.Search(context.Background(), v.ID, "install", "setup", "")
	if err != nil || len(results) != 1 {
		t.Fatalf("results=%#v err=%v", results, err)
	}
	other := filepath.Join(dir, "other")
	if err := os.MkdirAll(other, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(other, "index.md"), []byte("# CLI\n\nInstall widgets with the CLI."), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := (Importer{Store: store}).Run(context.Background(), Job{SourceDir: other, Product: "cli", Version: "2.0"}); err != nil {
		t.Fatal(err)
	}
	all, err := store.SearchAll(context.Background(), "install")
	if err != nil || len(all) != 2 || all[0].Product == "" || all[0].Version == "" {
		t.Fatalf("global results=%#v err=%v", all, err)
	}
}
