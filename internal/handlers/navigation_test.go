package handlers

import (
	"github.com/dockube/dockube/internal/models"
	"testing"
)

func TestOrderedNavigationHonorsConfiguredPages(t *testing.T) {
	docs := []models.Document{{Path: "guide/install", Title: "Install"}, {Path: "index", Title: "Home"}, {Path: "reference", Title: "Reference"}}
	got := orderedNavigation(docs, []string{"index.md", "guide/install.md"})
	if got[0].Path != "index" || got[1].Path != "guide/install" || got[2].Path != "reference" {
		t.Fatalf("navigation=%#v", got)
	}
}
