package main

import (
	"net/http"

	"blik/blikconfig"
	"blik/content"
	"blik/template"
)

func main() {
	cfg := parseConfig()

	blikCfg := blikconfig.NewStore(cfg.Root)
	blikCfg.Preload()
	tmpl := template.NewEngine(cfg.TemplateDir)
	h := content.NewHandler(cfg.Root, blikCfg, tmpl, cfg.IconsDir)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.Handle("/", h)

	hnd := http.Handler(mux)
	if cfg.ServerName != "" {
		hnd = serverHeaderMiddleware(hnd, cfg.ServerName)
	}

	serve(cfg, recoveryMiddleware(loggingMiddleware(hnd)))
}
