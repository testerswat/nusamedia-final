package main

import (
	"context"
	"fmt"
	"github.com/nusamedia/platform/apps/api/internal/config"
	"github.com/nusamedia/platform/apps/api/internal/httpapi"
	"github.com/nusamedia/platform/apps/api/internal/service"
	"github.com/nusamedia/platform/apps/api/internal/store"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func main() {
	c := config.Load()
	if c.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	st, e := store.Open(context.Background(), c.DatabaseURL)
	if e != nil {
		log.Fatal(e)
	}
	social := service.Social{DB: st.DB}
	if err := social.EnsurePlatformSettings(context.Background()); err != nil {
		log.Fatal(err)
	}
	r := httpapi.New(st.DB, social, c.JWTSecret)
	handler := r.Handler()
	webDir := os.Getenv("WEB_DIR")
	adminDir := os.Getenv("ADMIN_DIR")
	mux := http.NewServeMux()
	mux.Handle("/api/", handler)
	mux.HandleFunc("/health", func(w http.ResponseWriter, req *http.Request) { handler.ServeHTTP(w, req) })
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, req *http.Request) { handler.ServeHTTP(w, req) })
	mux.HandleFunc("/admin", serveSPA(adminDir, "/admin"))
	mux.HandleFunc("/admin/", serveSPA(adminDir, "/admin"))
	mux.HandleFunc("/", serveSPA(webDir, ""))
	log.Printf("NusaMedia full-stack engine listening on :%d", c.Port)
	srv := &http.Server{Addr: "0.0.0.0:" + itoa(c.Port), Handler: mux, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
func itoa(i int) string { return fmt.Sprintf("%d", i) }

func serveSPA(root, prefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if root == "" {
			http.NotFound(w, r)
			return
		}
		path := strings.TrimPrefix(r.URL.Path, prefix)
		clean := filepath.Clean(strings.TrimPrefix(path, "/"))
		if strings.HasPrefix(clean, "..") {
			http.NotFound(w, r)
			return
		}
		candidate := filepath.Join(root, clean)
		if path == "" || path == "/" || !fileExists(candidate) {
			candidate = filepath.Join(root, "index.html")
		}
		http.ServeFile(w, r, candidate)
	}
}
func fileExists(path string) bool { info, err := os.Stat(path); return err == nil && !info.IsDir() }
