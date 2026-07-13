package importer

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/dockube/dockube/internal/markdown"
	"github.com/dockube/dockube/internal/models"
)

type Job struct {
	SourceDir, Product, Title, Description, Version string
	Nav                                             []string
}
type Importer struct{ Store models.Store }

func (i Importer) Run(ctx context.Context, j Job) (int, error) {
	v, err := i.Store.EnsureProductVersionDetails(ctx, j.Product, j.Title, j.Description, j.Version)
	if err != nil {
		return 0, err
	}
	count := 0
	err = filepath.WalkDir(j.SourceDir, func(file string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			return nil
		}
		b, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(j.SourceDir, file)
		if err != nil {
			return err
		}
		path := strings.TrimSuffix(filepath.ToSlash(rel), filepath.Ext(rel))
		if path == "index" {
			path = ""
		}
		r, err := markdown.Render(string(b), j.Product, j.Version)
		if err != nil {
			return fmt.Errorf("%s: %w", rel, err)
		}
		err = i.Store.UpsertDocument(ctx, models.Document{VersionID: v.ID, Path: path, Title: r.Title, Owner: r.Owner, Tags: r.Tags, Source: string(b), HTML: r.HTML})
		if err == nil {
			count++
		}
		return err
	})
	return count, err
}

// GitSource is the future webhook/clone contract; no remote provider is enabled in v1.
type GitSource interface {
	Sync(context.Context) (string, error)
}
