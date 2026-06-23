package main

import (
	"crypto/sha512"
	"embed"
	"encoding/base64"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"blik/blikconfig"
	"blik/content"
	"blik/template"

	"github.com/fsnotify/fsnotify"
	"github.com/grimdork/climate/fx"
)

//go:embed template/static/blik.js
var staticFS embed.FS
var blikSRI string

func init() {
	mime.AddExtensionType(".ico", "image/x-icon")
	mime.AddExtensionType(".webp", "image/webp")
	mime.AddExtensionType(".avif", "image/avif")

	data, err := staticFS.ReadFile("template/static/blik.js")
	if err == nil {
		h := sha512.Sum384(data)
		blikSRI = "sha384-" + base64.StdEncoding.EncodeToString(h[:])
	}
}

func watchBlikFiles(root string, store *blikconfig.Store) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fx.Fprintln(os.Stderr, "{logstamp} {warning}config watcher:{@} unavailable — {}", err)
		return
	}
	defer watcher.Close()

	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err == nil && info.IsDir() {
			watcher.Add(path)
		}
		return nil
	})

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if strings.HasSuffix(event.Name, ".blik") {
				dir := filepath.Dir(event.Name)
				store.Invalidate(dir)
				fx.Println("{logstamp} {info}config:{@} reloaded {}", dir)
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			fx.Fprintln(os.Stderr, "{logstamp} {warning}config watcher:{@} {}", err)
		}
	}
}

func main() {
	cfg := parseConfig()

	blikCfg := blikconfig.NewStore(cfg.Root)
	blikCfg.Preload()
	go watchBlikFiles(cfg.Root, blikCfg)
	tmpl := template.NewEngine(cfg.TemplateDir)
	if blikSRI != "" {
		tmpl.SetSRI(blikSRI)
	}
	h := content.NewHandler(cfg.Root, blikCfg, tmpl, cfg.IconsDir)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/blik.js", func(w http.ResponseWriter, r *http.Request) {
		data, err := staticFS.ReadFile("template/static/blik.js")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.Write(data)
	})
	mux.Handle("/", h)

	rootCfg := blikCfg.GetConfig(cfg.Root)

	hnd := http.Handler(mux)
	if rootCfg.CSP != "" || rootCfg.HSTS != "" {
		hnd = securityHeadersMiddleware(hnd, rootCfg.CSP, rootCfg.HSTS)
	}
	if cfg.ServerName != "" {
		hnd = serverHeaderMiddleware(hnd, cfg.ServerName)
	}

	serve(cfg, recoveryMiddleware(loggingMiddleware(hnd)))
}
