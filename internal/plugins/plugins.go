package plugins

import "context"

type Document struct {
	Path, Source, HTML string
	Metadata           map[string]string
}
type DocProcessor interface {
	Name() string
	Process(context.Context, *Document) error
}
type Renderer interface {
	Name() string
	Render(string) (string, bool)
}
type SearchIndexer interface {
	Name() string
	Index(context.Context, Document) error
}
type AuthProvider interface {
	Name() string
	Authenticate(context.Context, string) (string, error)
}
type Registry struct {
	processors []DocProcessor
	renderers  []Renderer
	indexers   []SearchIndexer
}

func (r *Registry) RegisterProcessor(v DocProcessor) { r.processors = append(r.processors, v) }
func (r *Registry) RegisterRenderer(v Renderer)      { r.renderers = append(r.renderers, v) }
func (r *Registry) Renderers() []Renderer            { return append([]Renderer(nil), r.renderers...) }

// MermaidRenderer marks Mermaid code for client-side rendering. It does not execute user content.
type MermaidRenderer struct{}

func (MermaidRenderer) Name() string                        { return "mermaid" }
func (MermaidRenderer) Render(source string) (string, bool) { return source, false }
