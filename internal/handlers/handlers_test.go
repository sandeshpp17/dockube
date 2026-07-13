package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dockube/dockube/internal/db"
	"github.com/dockube/dockube/internal/importer"
	"github.com/dockube/dockube/internal/models"
)

func TestDocumentSearchAndCSRF(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	os.Mkdir(source, 0755)
	os.WriteFile(filepath.Join(source, "index.md"), []byte("# Dockube\n\nFind widgets here."), 0644)
	database, err := db.Open(context.Background(), filepath.Join(dir, "dockube.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	store := models.Store{DB: database}
	imp := importer.Importer{Store: store}
	job := importer.Job{SourceDir: source, Product: "dockube", Version: "1.0"}
	if _, err = imp.Run(context.Background(), job); err != nil {
		t.Fatal(err)
	}
	a := App{Store: store, Importer: imp, ImportJobs: []importer.Job{job}}
	h := a.Routes()
	req := httptest.NewRequest(http.MethodGet, "/docs/dockube/latest/", nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != 200 || !strings.Contains(res.Body.String(), "Dockube") {
		t.Fatalf("doc response: %d %s", res.Code, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "name=\"csrf\" value=\"") || len(res.Result().Cookies()) == 0 {
		t.Fatal("document page did not issue a CSRF cookie and form token")
	}
	req = httptest.NewRequest(http.MethodGet, "/search/dockube/latest?q=widgets", nil)
	req.Header.Set("HX-Request", "true")
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != 200 || !strings.Contains(res.Body.String(), "Dockube") {
		t.Fatalf("search response: %d %s", res.Code, res.Body.String())
	}
	req = httptest.NewRequest(http.MethodGet, "/search?q=widgets", nil)
	req.Header.Set("HX-Request", "true")
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != 200 || !strings.Contains(res.Body.String(), "dockube") {
		t.Fatalf("global search response: %d %s", res.Code, res.Body.String())
	}
	req = httptest.NewRequest(http.MethodGet, "/docs/dockube/latest", nil)
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusPermanentRedirect || res.Header().Get("Location") != "/docs/dockube/latest/" {
		t.Fatalf("canonical redirect: %d %q", res.Code, res.Header().Get("Location"))
	}
	req = httptest.NewRequest(http.MethodGet, "/docs/dockube/latest/missing", nil)
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusNotFound || !strings.Contains(res.Body.String(), "Page not found") {
		t.Fatalf("not found response: %d %s", res.Code, res.Body.String())
	}
	req = httptest.NewRequest(http.MethodGet, "/docs/dockube/latest/", nil)
	req.Header.Set("HX-Request", "true")
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != 200 || strings.Contains(res.Body.String(), "<!doctype html>") || !strings.Contains(res.Body.String(), "doc-layout") {
		t.Fatalf("htmx document fragment: %d %s", res.Code, res.Body.String())
	}
	req = httptest.NewRequest(http.MethodPost, "/admin/import", nil)
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("missing CSRF status = %d", res.Code)
	}
}
