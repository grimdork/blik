package content

import (
	"fmt"
	"html/template"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
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

const renderSuffix = "/render.gohtml"
const archiveSuffix = "/archive.gohtml"

type Handler struct {
	root      string
	blikStore *blikconfig.Store
	tmpl      *bliktmpl.Engine
}

func NewHandler(root string, bs *blikconfig.Store, tmpl *bliktmpl.Engine, iconDir string) *Handler {
	initIconCache(iconDir)
	return &Handler{
		root:      root,
		blikStore: bs,
		tmpl:      tmpl,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := filepath.Clean(r.URL.Path)

	if strings.HasSuffix(path, ".blik") {
		http.NotFound(w, r)
		return
	}

	fullPath := filepath.Join(h.root, path)
	if !strings.HasPrefix(fullPath, filepath.Clean(h.root)+string(filepath.Separator)) && fullPath != filepath.Clean(h.root) {
		http.NotFound(w, r)
		return
	}

	lfi, err := os.Lstat(fullPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	cfg := h.blikStore.GetConfig(filepath.Dir(fullPath))
	if !cfg.Symlinks && lfi.Mode()&os.ModeSymlink != 0 {
		http.NotFound(w, r)
		return
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	dir := filepath.Dir(fullPath)
	if info.IsDir() {
		dir = fullPath
	}
	cfg = h.blikStore.GetConfig(dir)

	if !info.IsDir() {
		name := filepath.Base(fullPath)
		_, wantsInfo := r.URL.Query()["info"]
		if !wantsInfo {
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
				h.serveInfo(w, r, fullPath, name, cfg)
				return
			}
		}

		http.ServeFile(w, r, fullPath)
		return
	}

	if !strings.HasSuffix(r.URL.Path, "/") {
		http.Redirect(w, r, r.URL.Path+"/", http.StatusMovedPermanently)
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

	result, err := render.Markdown(src)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	css, darkCSS, printCSS := h.tmpl.CSS("md")
	tmplName := "md" + renderSuffix
	if cfg.MarkdownTemplate != "" {
		tmplName = cfg.MarkdownTemplate + renderSuffix
	}

	out, err := h.tmpl.Render(tmplName, map[string]any{
		"Title":    name,
		"Content":  template.HTML(result.HTML),
		"Headings": result.Headings,
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

func (h *Handler) serveInfo(w http.ResponseWriter, r *http.Request, fullPath, name string, cfg *blikconfig.Config) {
	if strings.HasSuffix(name, ".zip") || strings.HasSuffix(name, ".tar.gz") || strings.HasSuffix(name, ".tgz") || strings.HasSuffix(name, ".tar") {
		h.serveArchiveInfo(w, r, fullPath, name, cfg)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	ext := strings.ToLower(filepath.Ext(name))
	if ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif" || ext == ".webp" || ext == ".bmp" || ext == ".ico" {
		h.serveImageInfo(w, r, fullPath, name)
		return
	}

	fmt.Fprintf(w, "<html><body><h1>%s</h1><p>No detailed information available.</p></body></html>", name)
}

func (h *Handler) serveImageInfo(w http.ResponseWriter, r *http.Request, fullPath, name string) {
	f, err := os.Open(fullPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()

	dim := "unknown"
	format := "unknown"
	if cfg, decName, err := image.DecodeConfig(f); err == nil {
		format = decName
		dim = fmt.Sprintf("%dx%d", cfg.Width, cfg.Height)
	}
	f.Seek(0, 0)

	if format == "unknown" {
		format = guessFormat(name)
	}

	fi, err := f.Stat()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, _ = fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>%s</title>
<style>
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;max-width:800px;margin:24px auto;padding:0 16px;background:#fff;color:#333}
img{max-width:100%%;border-radius:6px;box-shadow:0 2px 8px rgba(0,0,0,.1)}
table{width:100%%;border-collapse:collapse;margin-top:16px}
th,td{text-align:left;padding:8px 12px;border-bottom:1px solid #ddd}
th{font-weight:600;color:#555;width:120px}
h1{border-bottom:2px solid #ddd;padding-bottom:8px}
</style>
</head>
<body>
<h1>%s</h1>
<img src="?">
<table>
<tr><th>Format</th><td>%s</td></tr>
<tr><th>Dimensions</th><td>%s</td></tr>
<tr><th>File size</th><td>%s</td></tr>
</table>
</body>
</html>`, name, name, format, dim, formatSize(fi.Size()))
}

func guessFormat(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".jpg", ".jpeg":
		return "JPEG"
	case ".png":
		return "PNG"
	case ".gif":
		return "GIF"
	case ".webp":
		return "WebP"
	case ".bmp":
		return "BMP"
	case ".ico":
		return "ICO"
	}
	return ext
}

func (h *Handler) serveArchiveInfo(w http.ResponseWriter, r *http.Request, fullPath, name string, cfg *blikconfig.Config) {
	ainfo, err := archive.Read(fullPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	size := formatSize(ainfo.TotalSize)
	tmplName := "archive" + archiveSuffix
	if cfg.ArchiveTemplate != "" {
		tmplName = cfg.ArchiveTemplate + archiveSuffix
	}

	css, darkCSS, printCSS := h.tmpl.CSS("archive")
	out, err := h.tmpl.Render(tmplName, map[string]any{
		"FileName":  name,
		"Format":    ainfo.Format,
		"FileCount": ainfo.FileCount,
		"Size":      size,
		"Tree":      template.HTML(ainfo.TreeHTML),
		"CSS":       css,
		"DarkCSS":   darkCSS,
		"PrintCSS":  printCSS,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, out)
}

type dirEntry struct {
	Name      string
	Size      string
	ModTime   string
	IsDir     bool
	HasInfo   bool
	Icon      template.HTML
	Thumbnail string
}

func (h *Handler) serveDirectory(w http.ResponseWriter, r *http.Request, fullPath, urlPath string, cfg *blikconfig.Config) {
	dirents, err := os.ReadDir(fullPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var entries []dirEntry
	for _, de := range dirents {
		name := de.Name()
		if name == ".blik" || strings.HasSuffix(name, ".thumb") {
			continue
		}

		if !cfg.Symlinks && de.Type()&os.ModeSymlink != 0 {
			continue
		}

		fi, err := de.Info()
		if err != nil {
			continue
		}

		e := dirEntry{
			Name:    name,
			Size:    formatSize(fi.Size()),
			ModTime: fi.ModTime().Format(time.RFC822),
			IsDir:   fi.IsDir(),
			HasInfo: cfg.HasInfo(name),
			Icon:    template.HTML(iconSVG(name, fi.IsDir())),
		}

		if !fi.IsDir() && cfg.Thumbnails && isImage(name) && thumbExists(filepath.Join(fullPath, name)) {
			e.Thumbnail = name + ".thumb"
		}

		entries = append(entries, e)
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return entries[i].Name < entries[j].Name
	})

	css, darkCSS, printCSS := h.tmpl.CSS("webroot")
	out, err := h.tmpl.Render("webroot/listing.gohtml", map[string]any{
		"Title":    "Index of " + urlPath,
		"Entries":  entries,
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
