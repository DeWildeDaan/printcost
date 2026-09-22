package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"printcost/internal/api"
	"printcost/internal/db"
)

//go:embed all:web
var webFS embed.FS

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	dbPath := getenv("DB_PATH", "./printcost.db")
	port := getenv("PORT", "8080")
	basePrefix := strings.TrimSuffix(os.Getenv("BASE_URL_PREFIX"), "/")

	sqlDB, err := db.Open(dbPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer sqlDB.Close()

	if err := db.Seed(sqlDB); err != nil {
		log.Fatalf("failed to seed database: %v", err)
	}

	staticFS, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatalf("failed to load embedded web assets: %v", err)
	}
	fileServer := http.FileServer(http.FS(staticFS))

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Mount(basePrefix+"/api/v1", api.New(sqlDB).Routes())

	indexHTML, err := fs.ReadFile(staticFS, "index.html")
	if err != nil {
		log.Fatalf("failed to read embedded index.html: %v", err)
	}

	// Serve static assets when they exist; otherwise fall back to
	// index.html so the client-side router can handle the route.
	r.Get(basePrefix+"/*", func(w http.ResponseWriter, req *http.Request) {
		path := strings.TrimPrefix(req.URL.Path, basePrefix)
		path = strings.TrimPrefix(path, "/")
		if path != "" {
			if f, err := staticFS.Open(path); err == nil {
				f.Close()
				req.URL.Path = "/" + path
				fileServer.ServeHTTP(w, req)
				return
			}
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(indexHTML)
	})

	log.Printf("PrintCost listening on :%s (db=%s)", port, dbPath)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
