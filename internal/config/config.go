package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	Address, DBPath, SourceDir, Product, Version, CatalogPath string
	ImportOnStart                                             bool
}

func Load() Config {
	c := Config{Address: value("DOCKUBE_ADDR", ":8080"), DBPath: value("DOCKUBE_DB_PATH", filepath.Join("data", "dockube.db")), SourceDir: value("DOCKUBE_SOURCE_DIR", "./docs"), Product: value("DOCKUBE_PRODUCT", "dockube"), Version: value("DOCKUBE_VERSION", "latest"), CatalogPath: value("DOCKUBE_CONFIG", "dockube.yml"), ImportOnStart: value("DOCKUBE_IMPORT_ON_START", "true") != "false"}
	return c
}
func value(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
