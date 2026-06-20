package content

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"blik/archive"
	"blik/blikconfig"
	"blik/render"
	bliktmpl "blik/template"
)

type Handler struct {
	root      string
	blikStore *blikconfig.Store
	tmpl      *bliktmpl.Engine
}

func NewHandler(root string, bs *blikconfig.Store, tmpl *bliktmpl.Engine) *Handler {
	return &Handler{
		root:      root,
		blikStore: bs,
		tmpl:      tmpl,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := filepath.Clean(r.URL.Path)
	if strings.HasPrefix(path, "/..") || strings.Contains(path, "..") {
		http.NotFound(w, r)
		return
	}

	if strings.HasSuffix(path, ".blik") {
		http.NotFound(w, r)
		return
	}

	fullPath := filepath.Join(h.root, path)
	if !strings.HasPrefix(fullPath, filepath.Clean(h.root)+string(filepath.Separator)) && fullPath != filepath.Clean(h.root) {
		http.NotFound(w, r)
		return
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	dir := filepath.Dir(fullPath)
	cfg := h.blikStore.GetConfig(dir)

	if !info.IsDir() {
		name := filepath.Base(fullPath)
		if r.URL.Query().Get("info") == "" {
			switch cfg.MatchHandler(name) {
			case "markdown":
				h.serveMarkdown(w, r, fullPath, name, cfg)
				return
			case "archive":
				w.Header().Set("Content-Type", "application/octet-stream")
				http.ServeFile(w, r, fullPath)
				return
			}
		} else {
			if cfg.HasInfo(name) {
				h.serveInfo(w, r, fullPath, name)
				return
			}
		}

		http.ServeFile(w, r, fullPath)
		return
	}

	h.serveDirectory(w, r, fullPath, path, cfg)
}

func (h *Handler) serveMarkdown(w http.ResponseWriter, r *http.Request, fullPath, name string, cfg *blikconfig.Config) {
	src, err := os.ReadFile(fullPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	htmlContent, err := render.Markdown(src)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	css, darkCSS, printCSS := h.tmpl.CSS("markdown")
	tmplName := "markdown/render.gohtml"
	if cfg.MarkdownTemplate != "" {
		tmplName = cfg.MarkdownTemplate + "/render.gohtml"
	}

	out, err := h.tmpl.Render(tmplName, map[string]any{
		"Title":    name,
		"Content":  template.HTML(htmlContent),
		"CSS":      css,
		"DarkCSS":  darkCSS,
		"PrintCSS": printCSS,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, out)
}

func (h *Handler) serveInfo(w http.ResponseWriter, r *http.Request, fullPath, name string) {
	if strings.HasSuffix(name, ".zip") || strings.HasSuffix(name, ".tar.gz") || strings.HasSuffix(name, ".tgz") || strings.HasSuffix(name, ".tar") {
		h.serveArchiveInfo(w, r, fullPath, name)
		return
	}

	fmt.Fprintf(w, "<html><body><h1>%s</h1><p>No detailed information available.</p></body></html>", name)
}

func (h *Handler) serveArchiveInfo(w http.ResponseWriter, r *http.Request, fullPath, name string) {
	ainfo, err := archive.Read(fullPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	size := formatSize(ainfo.TotalSize)
	tmplName := "archive/archive.gohtml"
	out, err := h.tmpl.Render(tmplName, map[string]any{
		"FileName":  name,
		"Format":    ainfo.Format,
		"FileCount": ainfo.FileCount,
		"Size":      size,
		"Tree":      template.HTML(ainfo.TreeHTML),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	css, darkCSS, printCSS := h.tmpl.CSS("archive")
	if css != "" || darkCSS != "" || printCSS != "" {
		var b bytes.Buffer
		b.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n<meta charset=\"UTF-8\">\n<meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
		b.WriteString("<title>" + name + "</title>\n")
		if css != "" {
			b.WriteString("<style>" + css + "</style>\n")
		}
		if darkCSS != "" {
			b.WriteString("<style media=\"(prefers-color-scheme:dark)\">" + darkCSS + "</style>\n")
		}
		if printCSS != "" {
			b.WriteString("<style media=\"print\">" + printCSS + "</style>\n")
		}
		b.WriteString("</head>\n<body>\n")
		b.WriteString(out)
		b.WriteString("</body>\n</html>\n")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, b.String())
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, out)
}

type dirEntry struct {
	Name    string
	Size    string
	ModTime string
	IsDir   bool
	HasInfo bool
}

type listingData struct {
	Path    string
	Entries []dirEntry
}

func (h *Handler) serveDirectory(w http.ResponseWriter, r *http.Request, fullPath, urlPath string, cfg *blikconfig.Config) {
	f, err := os.Open(fullPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()

	names, err := f.Readdirnames(-1)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var entries []dirEntry
	for _, name := range names {
		if name == ".blik" {
			continue
		}

		fi, err := os.Stat(filepath.Join(fullPath, name))
		if err != nil {
			continue
		}

		e := dirEntry{
			Name:    name,
			Size:    formatSize(fi.Size()),
			ModTime: fi.ModTime().Format(time.RFC822),
			IsDir:   fi.IsDir(),
			HasInfo: cfg.HasInfo(name),
		}
		entries = append(entries, e)
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return entries[i].Name < entries[j].Name
	})

	out, err := h.tmpl.Render("default/listing.gohtml", listingData{
		Path:    urlPath,
		Entries: entries,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, out)
}

func formatSize(n int64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.1f GiB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MiB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KiB", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%d B", n)
	}
}
