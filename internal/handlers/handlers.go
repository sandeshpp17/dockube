package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"html/template"
	"net/http"
	"net/url"
	"strings"

	"github.com/dockube/dockube/internal/importer"
	"github.com/dockube/dockube/internal/models"
	"github.com/go-chi/chi/v5"
)

type App struct {
	Store      models.Store
	Importer   importer.Importer
	ImportJobs []importer.Job
	Navigation map[string][]string
}
type page struct {
	Product                 string
	Version                 models.Version
	Versions                []models.Version
	Document                models.Document
	Products                []models.Product
	Results                 []models.SearchResult
	Navigation              []models.Document
	Query, CSRF, ActivePath string
}

func docURL(product, version, path string) string {
	if path == "" || path == "index" {
		return "/docs/" + product + "/" + version + "/"
	}
	return "/docs/" + product + "/" + version + "/" + strings.TrimPrefix(path, "/")
}
func active(path, current string) string {
	if path == current {
		return "is-active"
	}
	return ""
}

var funcs = template.FuncMap{"safe": func(s string) template.HTML { return template.HTML(s) }, "docURL": docURL, "active": active}

var base = template.Must(template.New("base").Funcs(funcs).Parse(`<!doctype html><html lang="en" data-theme="light"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><script>(function(){var t=localStorage.getItem('dockube-theme');document.documentElement.dataset.theme=t||(matchMedia('(prefers-color-scheme: dark)').matches?'dark':'light')})()</script><title>Dockube</title><link rel="stylesheet" href="/static/app.css"><script defer src="/static/htmx.min.js"></script><script defer src="/static/app.js"></script></head><body hx-boost="true" hx-target="#content" hx-swap="innerHTML" hx-push-url="true"><div class="progress" id="progress"></div><header class="topbar"><button class="icon-button mobile-only" type="button" data-open="component-drawer" aria-label="Open components">☰</button><a class="brand" href="/" aria-label="Dockube home"><span class="brand-mark">D</span> Dockube</a><button class="search-launch" type="button" data-open="search-dialog"><span>⌕</span><span>Search documentation</span><kbd>/</kbd></button><div class="topbar-actions"><button class="icon-button" type="button" data-theme-toggle aria-label="Toggle color scheme">◐</button><form method="post" action="/admin/import"><input type="hidden" name="csrf" value="{{.CSRF}}"><button class="text-button">Refresh</button></form></div></header><div class="drawer-backdrop" data-close-drawers></div><aside class="drawer" id="component-drawer" aria-label="Components"><div class="drawer-header"><strong>Components</strong><button class="icon-button" data-close-drawers aria-label="Close">×</button></div>{{range .Products}}<a href="/docs/{{.Slug}}/latest/">{{.Name}}</a>{{end}}</aside><dialog class="search-dialog" id="search-dialog"><div class="search-panel"><div class="search-heading"><strong>Search documentation</strong><button class="icon-button" data-close-search aria-label="Close search">×</button></div><input id="global-search" name="q" autocomplete="off" placeholder="Search all components" hx-get="/search" hx-trigger="keyup changed delay:250ms" hx-target="#global-results" hx-indicator="#search-loading"><span id="search-loading" class="htmx-indicator">Searching…</span><div id="global-results"></div><p class="search-hint">Press <kbd>Esc</kbd> to close</p></div></dialog><main><aside class="component-nav desktop-only"><p class="nav-label">Components</p>{{range .Products}}<a href="/docs/{{.Slug}}/latest/">{{.Name}}</a>{{end}}</aside><section id="content" tabindex="-1">{{template "content" .}}</section></main></body></html>{{define "content"}}<section class="landing"><p class="eyebrow">Documentation portal</p><h1>Everything your team needs to build.</h1><p>Choose a component to explore versioned technical documentation.</p><div class="component-cards">{{range .Products}}<a class="component-card" href="/docs/{{.Slug}}/latest/"><strong>{{.Name}}</strong><span>{{.Description}}</span><span aria-hidden="true">→</span></a>{{end}}</div></section>{{end}}`))
var document = template.Must(template.Must(base.Clone()).Parse(`{{define "content"}}<div class="doc-layout"><aside class="page-nav desktop-only"><p class="nav-label">{{.Product}}</p><select class="version-select" aria-label="Version" data-version-select data-product="{{.Product}}">{{range .Versions}}<option value="{{.Version}}" {{if eq .Version $.Version.Version}}selected{{end}}>{{.Version}}</option>{{end}}</select><nav>{{range .Navigation}}<a class="{{active .Path $.ActivePath}}" href="{{docURL $.Product $.Version.Version .Path}}">{{.Title}}</a>{{end}}</nav></aside><aside class="drawer page-drawer" id="page-drawer" aria-label="Page navigation"><div class="drawer-header"><strong>{{.Product}}</strong><button class="icon-button" data-close-drawers aria-label="Close">×</button></div>{{range .Navigation}}<a class="{{active .Path $.ActivePath}}" href="{{docURL $.Product $.Version.Version .Path}}">{{.Title}}</a>{{end}}</aside><article class="doc-content"><div class="doc-toolbar"><button class="icon-button mobile-only" data-open="page-drawer" aria-label="Open page navigation">☰</button><div class="breadcrumbs"><a href="/">Components</a><span>/</span><a href="/docs/{{.Product}}/{{.Version.Version}}/">{{.Product}}</a><span>/</span><span>{{.Document.Title}}</span></div></div><div class="component-search"><input name="q" placeholder="Search {{.Product}}" hx-get="/search/{{.Product}}/{{.Version.Version}}" hx-trigger="keyup changed delay:300ms" hx-target="#component-results"><div id="component-results"></div></div><div class="markdown-body"><h1>{{.Document.Title}}</h1>{{if .Document.Owner}}<p class="meta">Owner: {{.Document.Owner}}</p>{{end}}<div>{{safe .Document.HTML}}</div></div></article></div>{{end}}`))
var productsPartial = template.Must(template.New("products").Funcs(funcs).Parse(`{{range .Products}}<a class="component-card" href="/docs/{{.Slug}}/latest/"><strong>{{.Name}}</strong><span>{{.Description}}</span><span aria-hidden="true">→</span></a>{{else}}<p>No components have been imported yet.</p>{{end}}`))
var resultsPartial = template.Must(template.New("results").Funcs(funcs).Parse(`{{if .Results}}<div class="search-results">{{range .Results}}<a href="{{docURL .Product .Version .Path}}"><strong>{{.Title}}</strong>{{if .Product}}<span class="result-scope">{{.Product}} · {{.Version}}</span>{{end}}<small>{{.Snippet}}</small></a>{{end}}</div>{{else if .Query}}<p class="empty-state">No matches found.</p>{{end}}`))
var notFound = template.Must(template.Must(base.Clone()).Parse(`{{define "content"}}<section class="not-found"><p class="eyebrow">404</p><h1>Page not found</h1><p>The requested documentation page does not exist in this component and version.</p><a class="primary-button" href="/">Back to components</a></section>{{end}}`))

func (a App) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(csrf)
	r.Get("/", a.Home)
	r.Get("/docs/{product}/{version}", a.CanonicalDoc)
	r.Get("/docs/{product}/{version}/", a.Doc)
	r.Get("/docs/{product}/{version}/*", a.Doc)
	r.Get("/search", a.GlobalSearch)
	r.Get("/search/{product}/{version}", a.Search)
	r.Post("/admin/import", a.Import)
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
	r.NotFound(a.NotFound)
	return r
}
func (a App) CanonicalDoc(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, r.URL.Path+"/", http.StatusPermanentRedirect)
}
func (a App) Home(w http.ResponseWriter, r *http.Request) {
	p := a.makePage(r.Context(), r)
	if partial(r) {
		_ = productsPartial.Execute(w, p)
		return
	}
	_ = base.Execute(w, p)
}
func (a App) Doc(w http.ResponseWriter, r *http.Request) {
	product, ver := chi.URLParam(r, "product"), chi.URLParam(r, "version")
	v, err := a.Store.ResolveVersion(r.Context(), product, ver)
	if err != nil {
		a.NotFound(w, r)
		return
	}
	path := strings.TrimPrefix(chi.URLParam(r, "*"), "/")
	if path == "index" {
		http.Redirect(w, r, docURL(product, ver, ""), http.StatusPermanentRedirect)
		return
	}
	d, err := a.Store.Document(r.Context(), v.ID, path)
	if err != nil && path == "" {
		d, err = a.Store.Document(r.Context(), v.ID, "index")
	}
	if err != nil {
		a.NotFound(w, r)
		return
	}
	p := a.makePage(r.Context(), r)
	activePath := path
	if activePath == "" {
		activePath = d.Path
	}
	p.Product, p.Version, p.Document, p.ActivePath = product, v, d, activePath
	p.Versions, _ = a.Store.Versions(r.Context(), product)
	docs, _ := a.Store.Documents(r.Context(), v.ID)
	p.Navigation = orderedNavigation(docs, a.Navigation[product+"@"+v.Version])
	if partial(r) {
		_ = document.ExecuteTemplate(w, "content", p)
		return
	}
	_ = document.Execute(w, p)
}
func (a App) Search(w http.ResponseWriter, r *http.Request) {
	product, ver := chi.URLParam(r, "product"), chi.URLParam(r, "version")
	v, err := a.Store.ResolveVersion(r.Context(), product, ver)
	if err != nil {
		a.NotFound(w, r)
		return
	}
	p := a.makePage(r.Context(), r)
	p.Product, p.Version, p.Query = product, v, r.URL.Query().Get("q")
	p.Results, err = a.Store.Search(r.Context(), v.ID, p.Query, r.URL.Query().Get("tag"), r.URL.Query().Get("owner"))
	if err != nil {
		http.Error(w, "search failed", 500)
		return
	}
	_ = resultsPartial.Execute(w, p)
}
func (a App) GlobalSearch(w http.ResponseWriter, r *http.Request) {
	p := a.makePage(r.Context(), r)
	p.Query = r.URL.Query().Get("q")
	var err error
	p.Results, err = a.Store.SearchAll(r.Context(), p.Query)
	if err != nil {
		http.Error(w, "search failed", 500)
		return
	}
	_ = resultsPartial.Execute(w, p)
}
func (a App) Import(w http.ResponseWriter, r *http.Request) {
	if !validCSRF(r) {
		http.Error(w, "invalid CSRF token", http.StatusForbidden)
		return
	}
	go func() {
		for _, job := range a.ImportJobs {
			_, _ = a.Importer.Run(context.Background(), job)
		}
	}()
	w.Header().Set("HX-Redirect", "/")
	w.WriteHeader(http.StatusAccepted)
}
func (a App) NotFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	p := a.makePage(r.Context(), r)
	if partial(r) {
		_ = notFound.ExecuteTemplate(w, "content", p)
		return
	}
	_ = notFound.Execute(w, p)
}
func (a App) makePage(ctx context.Context, r *http.Request) page {
	ps, _ := a.Store.Products(ctx)
	return page{Products: ps, CSRF: csrfToken(r)}
}
func partial(r *http.Request) bool { return r.Header.Get("HX-Request") == "true" }
func csrf(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := r.Cookie("dockube_csrf"); err != nil {
			b := make([]byte, 24)
			_, _ = rand.Read(b)
			c := &http.Cookie{Name: "dockube_csrf", Value: base64.RawURLEncoding.EncodeToString(b), Path: "/", SameSite: http.SameSiteLaxMode, HttpOnly: true}
			http.SetCookie(w, c)
			r.AddCookie(c)
		}
		next.ServeHTTP(w, r)
	})
}
func csrfToken(r *http.Request) string {
	c, _ := r.Cookie("dockube_csrf")
	if c == nil {
		return ""
	}
	return c.Value
}
func validCSRF(r *http.Request) bool {
	c, _ := r.Cookie("dockube_csrf")
	return c != nil && r.FormValue("csrf") != "" && r.FormValue("csrf") == c.Value && sameOrigin(r)
}
func sameOrigin(r *http.Request) bool {
	if o := r.Header.Get("Origin"); o != "" {
		u, e := url.Parse(o)
		return e == nil && u.Host == r.Host
	}
	return true
}
func orderedNavigation(documents []models.Document, configured []string) []models.Document {
	if len(configured) == 0 {
		return documents
	}
	byPath := make(map[string]models.Document, len(documents))
	for _, d := range documents {
		byPath[d.Path] = d
	}
	ordered := make([]models.Document, 0, len(documents))
	used := make(map[string]bool)
	for _, path := range configured {
		path = strings.TrimSuffix(strings.TrimPrefix(path, "/"), ".md")
		if d, ok := byPath[path]; ok {
			ordered = append(ordered, d)
			used[path] = true
		}
	}
	for _, d := range documents {
		if !used[d.Path] {
			ordered = append(ordered, d)
		}
	}
	return ordered
}
