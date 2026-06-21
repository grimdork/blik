package main

import (
	"crypto/sha512"
	"embed"
	"encoding/base64"
	"mime"
	"net/http"

	"blik/blikconfig"
	"blik/content"
	"blik/template"
)

//go:embed template/static/blik.js
var staticFS embed.FS
var blikSRI string

func init() {
	mime.AddExtensionType(".ico", "image/x-icon")

	data, err := staticFS.ReadFile("template/static/blik.js")
	if err == nil {
		h := sha512.Sum384(data)
		blikSRI = "sha384-" + base64.StdEncoding.EncodeToString(h[:])
	}
}

func main() {
	cfg := parseConfig()

	blikCfg := blikconfig.NewStore(cfg.Root)
	blikCfg.Preload()
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
