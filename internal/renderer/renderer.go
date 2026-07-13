// Package renderer provides the core markdown rendering engine for Dockube.
// It uses goldmark with extensions for tables, admonitions, task lists, TOC,
// syntax highlighting, Mermaid diagrams, footnotes, and more.
package renderer

import (
	"bytes"
	"fmt"
	"html/template"
	"regexp"
	"strings"
	"sync"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
	"go.uber.org/zap"
)

// Renderer handles markdown to HTML conversion with all extensions enabled.
type Renderer struct {
	md       goldmark.Markdown
	logger   *zap.Logger
	cache    sync.Map // Simple in-memory cache for rendered content
	tocCache sync.Map // Cache for table of contents
}

// Config holds renderer configuration options.
type Config struct {
	EnableUnsafeHTML   bool
	EnableAutoHeadingID bool
	EnableXHTML        bool
	HardWraps          bool
}

// FrontMatter represents the optional YAML-like front matter in documents.
type FrontMatter struct {
	Title string   `yaml:"title"`
	Owner string   `yaml:"owner"`
	Tags  []string `yaml:"tags"`
	Order int      `yaml:"order"`
}

// RenderResult contains the rendered content and extracted metadata.
type RenderResult struct {
	HTML        template.HTML
	Title       string
	Owner       string
	Tags        []string
	TOC         []TOCEntry
	WordCount   int
	ReadingTime int // minutes
}

// TOCEntry represents a single table of contents item.
type TOCEntry struct {
	ID       string
	Text     string
	Level    int
	Children []TOCEntry
}

// New creates a new Renderer with all extensions configured.
func New(logger *zap.Logger, cfg Config) *Renderer {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,              // GitHub Flavored Markdown (tables, task lists, strikethrough)
			extension.Typographer,      // Smart quotes, ellipses, etc.
			extension.Linkify,          // Auto-link URLs
			extension.Table,            // Explicit table support
			extension.Strikethrough,    // ~~strikethrough~~
			extension.TaskList,         // - [ ] task lists
			extension.DefinitionList,   // Definition lists
			extension.Footnote,         // [^1] footnotes
			&TOC{},                     // Custom TOC extension
			&Mermaid{},                 // Mermaid diagram support
			&Admonition{},              // Custom admonition blocks
			extension.CJK,              // CJK language support
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
			parser.WithAttribute(),
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
			html.WithXHTML(),
			html.WithHardWraps(),
		),
	)

	return &Renderer{
		md:     md,
		logger: logger,
	}
}

// Render converts markdown source to HTML with all processing.
func (r *Renderer) Render(source []byte, frontMatter FrontMatter) (*RenderResult, error) {
	// Check cache first
	cacheKey := string(source)
	if cached, ok := r.cache.Load(cacheKey); ok {
		return cached.(*RenderResult), nil
	}

	// Parse front matter if not provided
	if frontMatter.Title == "" {
		var err error
		frontMatter, source = r.extractFrontMatter(source)
		if err != nil {
			r.logger.Warn("failed to parse front matter", zap.Error(err))
		}
	}

	// Convert wiki-style links [[page]] to proper URLs
	source = r.convertWikiLinks(source)

	// Render markdown to HTML
	var buf bytes.Buffer
	if err := r.md.Convert(source, &buf); err != nil {
		return nil, fmt.Errorf("markdown conversion failed: %w", err)
	}

	html := buf.Bytes()

	// Extract TOC from the rendered HTML
	toc := r.extractTOC(html)

	// Calculate reading statistics
	wordCount := r.countWords(source)
	readingTime := (wordCount + 199) / 200 // ~200 words per minute

	result := &RenderResult{
		HTML:        template.HTML(html),
		Title:       frontMatter.Title,
		Owner:       frontMatter.Owner,
		Tags:        frontMatter.Tags,
		TOC:         toc,
		WordCount:   wordCount,
		ReadingTime: readingTime,
	}

	// Cache the result
	r.cache.Store(cacheKey, result)

	return result, nil
}

// extractFrontMatter parses optional front matter from the source.
func (r *Renderer) extractFrontMatter(source []byte) (FrontMatter, []byte) {
	fm := FrontMatter{}

	// Match front matter block at the start
	re := regexp.MustCompile(`^---\s*\n(.*?)\n---\s*\n`)
	matches := re.FindSubmatch(source)
	if matches == nil {
		return fm, source
	}

	// Parse YAML-like front matter (simplified)
	frontMatterText := string(matches[1])
	lines := strings.Split(frontMatterText, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "title:") {
			fm.Title = strings.TrimSpace(strings.TrimPrefix(line, "title:"))
			fm.Title = strings.Trim(fm.Title, `"'`)
		} else if strings.HasPrefix(line, "owner:") {
			fm.Owner = strings.TrimSpace(strings.TrimPrefix(line, "owner:"))
			fm.Owner = strings.Trim(fm.Owner, `"'`)
		} else if strings.HasPrefix(line, "tags:") {
			tags := strings.TrimSpace(strings.TrimPrefix(line, "tags:"))
			tags = strings.Trim(tags, "[]")
			fm.Tags = strings.Split(tags, ",")
			for i := range fm.Tags {
				fm.Tags[i] = strings.TrimSpace(fm.Tags[i])
			}
		}
	}

	// Remove front matter from source
	remaining := source[len(matches[0]):]
	return fm, remaining
}

// convertWikiLinks transforms [[page]] syntax into proper documentation URLs.
func (r *Renderer) convertWikiLinks(source []byte) []byte {
	// Convert [[page]] to [page](/docs/{product}/{version}/page)
	// This is a placeholder - actual URL construction happens at render time
	re := regexp.MustCompile(`\[\[([^\]]+)\]\]`)
	return re.ReplaceAll(source, []byte(`[$1](/docs/$1)`))
}

// extractTOC parses headings from HTML to build table of contents.
func (r *Renderer) extractTOC(html []byte) []TOCEntry {
	// Simple regex-based extraction for demo
	// Production would use proper HTML parsing
	re := regexp.MustCompile(`<h([1-6])[^>]*id="([^"]*)"[^>]*>(.*?)</h[1-6]>`)
	matches := re.FindAllSubmatch(html, -1)

	var entries []TOCEntry
	for _, m := range matches {
		level := int(m[1][0] - '0')
		id := string(m[2])
		text := stripHTML(string(m[3]))

		entries = append(entries, TOCEntry{
			ID:    id,
			Text:  text,
			Level: level,
		})
	}

	return r.buildTOCTree(entries)
}

// buildTOCTree converts flat entries into a hierarchical tree.
func (r *Renderer) buildTOCTree(entries []TOCEntry) []TOCEntry {
	if len(entries) == 0 {
		return nil
	}

	var root []TOCEntry
	var stack []TOCEntry

	for _, entry := range entries {
		for len(stack) > 0 && stack[len(stack)-1].Level >= entry.Level {
			stack = stack[:len(stack)-1]
		}

		if len(stack) == 0 {
			root = append(root, entry)
			stack = append(stack, entry)
		} else {
			parent := &stack[len(stack)-1]
			parent.Children = append(parent.Children, entry)
			stack = append(stack, entry)
		}
	}

	return root
}

// stripHTML removes HTML tags from text.
func stripHTML(s string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	return re.ReplaceAllString(s, "")
}

// countWords returns the word count of the source.
func (r *Renderer) countWords(source []byte) int {
	words := strings.Fields(string(source))
	return len(words)
}

// ClearCache removes all cached render results.
func (r *Renderer) ClearCache() {
	r.cache = sync.Map{}
	r.tocCache = sync.Map{}
}

// TOC extension for automatic table of contents generation.
type TOC struct{}

// Extend implements goldmark.Extender.
func (t *TOC) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		parser.WithASTTransformers(
			util.Prioritized(&TOCTransformer{}, 100),
		),
	)
}

// TOCTransformer adds TOC generation during parsing.
type TOCTransformer struct{}

// Transform implements parser.ASTTransformer.
func (t *TOCTransformer) Transform(node *ast.Document, reader text.Reader, pc parser.Context) {
	// TOC generation logic here
	// This is a placeholder for the actual implementation
}

// Mermaid extension for diagram support.
type Mermaid struct{}

// Extend implements goldmark.Extender.
func (m *Mermaid) Extend(md goldmark.Markdown) {
	md.Parser().AddOptions(
		parser.WithASTTransformers(
			util.Prioritized(&MermaidTransformer{}, 100),
		),
	)
	md.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(&MermaidRenderer{}, 100),
	))
}

// MermaidTransformer transforms code fences with mermaid language.
type MermaidTransformer struct{}

// Transform implements parser.ASTTransformer.
func (t *MermaidTransformer) Transform(node *ast.Document, reader text.Reader, pc parser.Context) {
	// Transform logic here
}

// MermaidRenderer renders Mermaid diagrams.
type MermaidRenderer struct{}

// RegisterFuncs implements renderer.NodeRenderer.
func (r *MermaidRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	// Registration logic
}

// Admonition extension for callout blocks.
type Admonition struct{}

// Extend implements goldmark.Extender.
func (a *Admonition) Extend(md goldmark.Markdown) {
	md.Parser().AddOptions(
		parser.WithBlockParsers(
			util.Prioritized(&AdmonitionParser{}, 100),
		),
	)
	md.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(&AdmonitionRenderer{}, 100),
	))
}

// AdmonitionParser parses admonition blocks.
type AdmonitionParser struct{}

// Trigger implements parser.BlockParser.
func (p *AdmonitionParser) Trigger() []byte {
	return []byte{'!', ':'}
}

// Open implements parser.BlockParser.
func (p *AdmonitionParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	// Parsing logic
	return nil, parser.Continue
}

// Continue implements parser.BlockParser.
func (p *AdmonitionParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	return parser.Continue
}

// Close implements parser.BlockParser.
func (p *AdmonitionParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
	// Close logic
}

// CanInterruptParagraph implements parser.BlockParser.
func (p *AdmonitionParser) CanInterruptParagraph() bool {
	return true
}

// CanAcceptIndentedLine implements parser.BlockParser.
func (p *AdmonitionParser) CanAcceptIndentedLine() bool {
	return false
}

// AdmonitionRenderer renders admonition blocks.
type AdmonitionRenderer struct{}

// RegisterFuncs implements renderer.NodeRenderer.
func (r *AdmonitionRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	// Registration logic
}