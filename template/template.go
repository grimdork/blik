package template

import (
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Engine struct {
	dir   string
	mu    sync.RWMutex
	cache map[string]*template.Template
}

func NewEngine(dir string) *Engine {
	return &Engine{
		dir:   dir,
		cache: make(map[string]*template.Template),
	}
}

func (e *Engine) Render(name string, data any) (string, error) {
	tmpl, err := e.load(name)
	if err != nil {
		return "", err
	}
	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (e *Engine) CSS(name string) (string, string, string) {
	return e.readCSS(name + "/style.css"),
		e.readCSS(name + "/dark.css"),
		e.readCSS(name + "/print.css")
}

func (e *Engine) readCSS(path string) string {
	if e.dir == "" {
		return ""
	}
	p := filepath.Join(e.dir, path)
	b, err := os.ReadFile(p)
	if err != nil {
		return ""
	}
	return string(b)
}

func (e *Engine) load(name string) (*template.Template, error) {
	e.mu.RLock()
	t, ok := e.cache[name]
	e.mu.RUnlock()
	if ok {
		return t, nil
	}

	t = template.New(name)
	if e.dir != "" {
		p := filepath.Join(e.dir, name)
		if _, err := os.Stat(p); err == nil {
			t, err = t.ParseFiles(p)
			if err == nil {
				e.mu.Lock()
				e.cache[name] = t
				e.mu.Unlock()
				return t, nil
			}
		}
	}

	builtin, ok := builtinTemplates[name]
	if ok {
		t, err := t.Parse(builtin)
		if err != nil {
			return nil, err
		}
		e.mu.Lock()
		e.cache[name] = t
		e.mu.Unlock()
		return t, nil
	}

	return nil, os.ErrNotExist
}

var builtinTemplates = map[string]string{
	"default/listing.gohtml": listingTemplate,
	"markdown/render.gohtml": markdownTemplate,
	"archive/archive.gohtml": archiveTemplate,
}

const listingTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Index of {{.Path}}</title>
<style>
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;max-width:800px;margin:24px auto;padding:0 16px;background:#fff;color:#333}
h1{font-size:1.2rem;font-weight:600;margin-bottom:16px;padding-bottom:8px;border-bottom:1px solid #ddd}
table{width:100%;border-collapse:collapse}
th,td{text-align:left;padding:4px 8px}
th{border-bottom:2px solid #ddd;font-weight:600;font-size:.85rem;color:#555}
td{border-bottom:1px solid #eee}
tr:hover td{background:#f5f5f5}
a{color:#1967d2;text-decoration:none}
a:hover{text-decoration:underline}
.size{text-align:right;font-size:.85rem;color:#555;white-space:nowrap}
.time{font-size:.85rem;color:#555;white-space:nowrap}
.info{text-align:center}
.info a{padding:2px 8px;border:1px solid #ccc;border-radius:4px;font-size:.75rem;color:#555;cursor:pointer}
.info a:hover{background:#eee;text-decoration:none}
</style>
</head>
<body>
<h1>Index of {{.Path}}</h1>
<table>
<tr><th>Name</th><th>Size</th><th>Modified</th><th></th></tr>
{{range .Entries}}
<tr>
<td>{{if .IsDir}}<a href="{{.Name}}/">{{.Name}}/</a>{{else}}<a href="{{.Name}}">{{.Name}}</a>{{end}}</td>
<td class="size">{{.Size}}</td>
<td class="time">{{.ModTime}}</td>
<td class="info">{{if .HasInfo}}<a href="{{.Name}}?info">i</a>{{end}}</td>
</tr>
{{end}}
</table>
</body>
</html>`

const markdownTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>{{.Title}}</title>
{{if .CSS}}<style>{{.CSS}}</style>{{end}}
{{if .DarkCSS}}<style id="dark-css" media="(prefers-color-scheme:dark)">{{.DarkCSS}}</style>{{end}}
{{if .PrintCSS}}<style media="print">{{.PrintCSS}}</style>{{end}}
</head>
<body>
<div class="container">{{.Content}}</div>
</body>
</html>`

const archiveTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>{{.FileName}}</title>
<style>
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;max-width:800px;margin:24px auto;padding:0 16px;background:#fff;color:#333}
h1{font-size:1.2rem;font-weight:600}
.meta{display:flex;gap:24px;margin:16px 0;padding:12px;background:#f5f5f5;border-radius:8px}
.meta-item{font-size:.85rem}
.meta-item strong{display:block;font-size:.75rem;color:#555}
details{margin:4px 0}
details summary{cursor:pointer;padding:2px 4px;border-radius:4px;font-size:.9rem}
details summary:hover{background:#eee}
.file{display:flex;justify-content:space-between;padding:2px 4px 2px 20px;font-size:.85rem}
.file span{color:#555}
.dir{font-weight:500}
.dir::before{content:"▸ ";color:#888}
details[open]>.dir::before{content:"▾ "}
</style>
</head>
<body>
<h1>{{.FileName}}</h1>
<div class="meta">
<div class="meta-item"><strong>Format</strong>{{.Format}}</div>
<div class="meta-item"><strong>Entries</strong>{{.FileCount}}</div>
<div class="meta-item"><strong>Size</strong>{{.Size}}</div>
</div>
{{.Tree}}
</body>
</html>`
