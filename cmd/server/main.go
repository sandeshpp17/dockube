package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dockube/dockube/internal/catalog"
	"github.com/dockube/dockube/internal/config"
	"github.com/dockube/dockube/internal/db"
	"github.com/dockube/dockube/internal/handlers"
	"github.com/dockube/dockube/internal/importer"
	"github.com/dockube/dockube/internal/models"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()
	database, err := db.Open(ctx, cfg.DBPath)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()
	imp := importer.Importer{Store: models.Store{DB: database}}
	jobs, err := importJobs(cfg)
	if err != nil {
		log.Fatal(err)
	}
	if cfg.ImportOnStart {
		go func() {
			for _, job := range jobs {
				if _, err := imp.Run(context.Background(), job); err != nil {
					log.Printf("initial import: %v", err)
				}
			}
		}()
	}
	navigation := make(map[string][]string, len(jobs))
	for _, job := range jobs {
		navigation[job.Product+"@"+job.Version] = job.Nav
	}
	app := handlers.App{Store: models.Store{DB: database}, Importer: imp, ImportJobs: jobs, Navigation: navigation}
	srv := &http.Server{Addr: cfg.Address, Handler: app.Routes(), ReadHeaderTimeout: 5 * time.Second}
	go func() {
		log.Printf("Dockube listening on %s", cfg.Address)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	stop, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(stop)
}

func importJobs(cfg config.Config) ([]importer.Job, error) {
	if _, err := os.Stat(cfg.CatalogPath); err == nil {
		c, err := catalog.Load(cfg.CatalogPath)
		if err != nil {
			return nil, err
		}
		jobs := make([]importer.Job, 0, len(c.Content.Sources))
		for _, s := range c.Content.Sources {
			jobs = append(jobs, importer.Job{SourceDir: s.StartPath, Product: s.Slug(), Title: s.Title, Version: s.Version, Nav: s.Nav})
		}
		return jobs, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	return []importer.Job{{SourceDir: cfg.SourceDir, Product: cfg.Product, Version: cfg.Version}}, nil
}
