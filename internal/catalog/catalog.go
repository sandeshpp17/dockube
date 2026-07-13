// Package catalog loads Dockube's Antora-inspired YAML content catalog.
package catalog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Catalog struct {
	Site struct {
		Title string `yaml:"title"`
	} `yaml:"site"`
	Content struct {
		Sources []Source `yaml:"sources"`
	} `yaml:"content"`
}
type Source struct {
	URL       string   `yaml:"url"`
	Component string   `yaml:"component"`
	Title     string   `yaml:"title"`
	Version   string   `yaml:"version"`
	StartPath string   `yaml:"start_path"`
	Nav       []string `yaml:"nav"`
}
type antora struct {
	Name, Title, Version string
	Nav                  []string `yaml:"nav"`
}

func Load(path string) (Catalog, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Catalog{}, err
	}
	var c Catalog
	if err := yaml.Unmarshal(b, &c); err != nil {
		return c, fmt.Errorf("parse %s: %w", path, err)
	}
	base := filepath.Dir(path)
	for i := range c.Content.Sources {
		s := &c.Content.Sources[i]
		if s.URL == "" {
			return c, fmt.Errorf("content.sources[%d]: url is required", i)
		}
		if !filepath.IsAbs(s.URL) {
			s.URL = filepath.Join(base, s.URL)
		}
		if err := s.applyAntora(); err != nil {
			return c, err
		}
		if s.Component == "" {
			return c, fmt.Errorf("content.sources[%d]: component/name is required", i)
		}
		if s.Version == "" {
			s.Version = "latest"
		}
		if s.Title == "" {
			s.Title = s.Component
		}
		if s.StartPath == "" {
			s.StartPath = "."
		}
		s.StartPath = filepath.Join(s.URL, s.StartPath)
	}
	if len(c.Content.Sources) == 0 {
		return c, fmt.Errorf("%s has no content.sources", path)
	}
	return c, nil
}
func (s *Source) applyAntora() error {
	b, err := os.ReadFile(filepath.Join(s.URL, "antora.yml"))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var a antora
	if err = yaml.Unmarshal(b, &a); err != nil {
		return fmt.Errorf("parse %s: %w", filepath.Join(s.URL, "antora.yml"), err)
	}
	if s.Component == "" {
		s.Component = a.Name
	}
	if s.Title == "" {
		s.Title = a.Title
	}
	if s.Version == "" {
		s.Version = a.Version
	}
	if len(s.Nav) == 0 {
		s.Nav = a.Nav
	}
	if s.StartPath == "" {
		candidate := filepath.Join(s.URL, "modules", "ROOT", "pages")
		if info, e := os.Stat(candidate); e == nil && info.IsDir() {
			s.StartPath = filepath.Join("modules", "ROOT", "pages")
		}
	}
	return nil
}
func (s Source) Slug() string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(s.Component), " ", "-"))
}
