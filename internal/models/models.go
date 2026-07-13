package models

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type Store struct{ DB *sql.DB }
type Product struct {
	ID                      int64
	Slug, Name, Description string
}
type Version struct {
	ID, ProductID int64
	Version       string
}
type Document struct {
	ID, VersionID                    int64
	Path, Title, Owner, Source, HTML string
	Tags                             []string
}
type SearchResult struct{ Path, Title, Snippet, Product, Version string }

func (s Store) EnsureProductVersionDetails(ctx context.Context, slug, name, description, version string) (Version, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return Version{}, err
	}
	defer tx.Rollback()
	if name == "" {
		name = strings.ReplaceAll(slug, "-", " ")
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO products(slug,name,description) VALUES(?,?,?) ON CONFLICT(slug) DO UPDATE SET name=excluded.name,description=excluded.description", slug, name, description); err != nil {
		return Version{}, err
	}
	var pid int64
	if err = tx.QueryRowContext(ctx, "SELECT id FROM products WHERE slug=?", slug).Scan(&pid); err != nil {
		return Version{}, err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO product_versions(product_id,version) VALUES(?,?) ON CONFLICT(product_id,version) DO NOTHING", pid, version); err != nil {
		return Version{}, err
	}
	var v Version
	if err = tx.QueryRowContext(ctx, "SELECT id,product_id,version FROM product_versions WHERE product_id=? AND version=?", pid, version).Scan(&v.ID, &v.ProductID, &v.Version); err != nil {
		return Version{}, err
	}
	for _, a := range []string{"latest", "stable"} {
		if _, err = tx.ExecContext(ctx, "INSERT INTO version_aliases(product_version_id,alias) VALUES(?,?) ON CONFLICT(product_version_id,alias) DO UPDATE SET product_version_id=excluded.product_version_id", v.ID, a); err != nil {
			return Version{}, err
		}
	}
	return v, tx.Commit()
}

func (s Store) EnsureProductVersion(ctx context.Context, slug, version string) (Version, error) {
	return s.EnsureProductVersionDetails(ctx, slug, "", "", version)
}
func (s Store) Products(ctx context.Context) ([]Product, error) {
	rows, err := s.DB.QueryContext(ctx, "SELECT id,slug,name,description FROM products ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Product
	for rows.Next() {
		var p Product
		if err = rows.Scan(&p.ID, &p.Slug, &p.Name, &p.Description); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
func (s Store) ResolveVersion(ctx context.Context, product, name string) (Version, error) {
	var v Version
	err := s.DB.QueryRowContext(ctx, `SELECT pv.id,pv.product_id,pv.version FROM product_versions pv JOIN products p ON p.id=pv.product_id LEFT JOIN version_aliases a ON a.product_version_id=pv.id WHERE p.slug=? AND (pv.version=? OR a.alias=?) ORDER BY CASE WHEN pv.version=? THEN 0 ELSE 1 END LIMIT 1`, product, name, name, name).Scan(&v.ID, &v.ProductID, &v.Version)
	return v, err
}
func (s Store) Versions(ctx context.Context, product string) ([]Version, error) {
	rows, err := s.DB.QueryContext(ctx, "SELECT pv.id,pv.product_id,pv.version FROM product_versions pv JOIN products p ON p.id=pv.product_id WHERE p.slug=? ORDER BY pv.version DESC", product)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Version
	for rows.Next() {
		var v Version
		if err = rows.Scan(&v.ID, &v.ProductID, &v.Version); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s Store) Documents(ctx context.Context, versionID int64) ([]Document, error) {
	rows, err := s.DB.QueryContext(ctx, "SELECT id,product_version_id,path,title,owner,source,html FROM documents WHERE product_version_id=? ORDER BY path", versionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Document
	for rows.Next() {
		var d Document
		if err = rows.Scan(&d.ID, &d.VersionID, &d.Path, &d.Title, &d.Owner, &d.Source, &d.HTML); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
func (s Store) UpsertDocument(ctx context.Context, d Document) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	r, err := tx.ExecContext(ctx, "INSERT INTO documents(product_version_id,path,title,owner,source,html) VALUES(?,?,?,?,?,?) ON CONFLICT(product_version_id,path) DO UPDATE SET title=excluded.title,owner=excluded.owner,source=excluded.source,html=excluded.html,updated_at=CURRENT_TIMESTAMP", d.VersionID, d.Path, d.Title, d.Owner, d.Source, d.HTML)
	if err != nil {
		return err
	}
	id, err := r.LastInsertId()
	if err != nil {
		return err
	}
	if id == 0 {
		err = tx.QueryRowContext(ctx, "SELECT id FROM documents WHERE product_version_id=? AND path=?", d.VersionID, d.Path).Scan(&id)
		if err != nil {
			return err
		}
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM document_tags WHERE document_id=?", id); err != nil {
		return err
	}
	for _, tag := range d.Tags {
		if _, err = tx.ExecContext(ctx, "INSERT INTO document_tags(document_id,tag) VALUES(?,?)", id, tag); err != nil {
			return err
		}
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM documents_fts WHERE rowid=?", id); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO documents_fts(rowid,title,source,tags) VALUES(?,?,?,?)", id, d.Title, d.Source, strings.Join(d.Tags, " ")); err != nil {
		return err
	}
	return tx.Commit()
}
func (s Store) Document(ctx context.Context, versionID int64, path string) (Document, error) {
	var d Document
	err := s.DB.QueryRowContext(ctx, "SELECT id,product_version_id,path,title,owner,source,html FROM documents WHERE product_version_id=? AND path=?", versionID, path).Scan(&d.ID, &d.VersionID, &d.Path, &d.Title, &d.Owner, &d.Source, &d.HTML)
	if err != nil {
		return d, err
	}
	rows, err := s.DB.QueryContext(ctx, "SELECT tag FROM document_tags WHERE document_id=? ORDER BY tag", d.ID)
	if err != nil {
		return d, err
	}
	defer rows.Close()
	for rows.Next() {
		var t string
		rows.Scan(&t)
		d.Tags = append(d.Tags, t)
	}
	return d, rows.Err()
}
func (s Store) Search(ctx context.Context, versionID int64, q, tag, owner string) ([]SearchResult, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return nil, nil
	}
	query := `SELECT d.path,d.title,substr(d.source,1,180),'','' FROM documents_fts f JOIN documents d ON d.id=f.rowid WHERE d.product_version_id=? AND documents_fts MATCH ?`
	args := []any{versionID, q}
	if tag != "" {
		query += " AND EXISTS (SELECT 1 FROM document_tags t WHERE t.document_id=d.id AND t.tag=?)"
		args = append(args, tag)
	}
	if owner != "" {
		query += " AND d.owner=?"
		args = append(args, owner)
	}
	query += " LIMIT 20"
	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	defer rows.Close()
	var out []SearchResult
	for rows.Next() {
		var r SearchResult
		if err = rows.Scan(&r.Path, &r.Title, &r.Snippet, &r.Product, &r.Version); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// SearchAll queries every imported component and version. It intentionally does
// not resolve aliases so results retain their concrete documentation version.
func (s Store) SearchAll(ctx context.Context, q string) ([]SearchResult, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return nil, nil
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT d.path,d.title,substr(d.source,1,180),p.slug,pv.version FROM documents_fts f JOIN documents d ON d.id=f.rowid JOIN product_versions pv ON pv.id=d.product_version_id JOIN products p ON p.id=pv.product_id WHERE documents_fts MATCH ? ORDER BY p.name,pv.version DESC,d.title LIMIT 30`, q)
	if err != nil {
		return nil, fmt.Errorf("global search: %w", err)
	}
	defer rows.Close()
	var out []SearchResult
	for rows.Next() {
		var r SearchResult
		if err = rows.Scan(&r.Path, &r.Title, &r.Snippet, &r.Product, &r.Version); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
