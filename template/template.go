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
	"webroot/listing.gohtml": listingTemplate,
	"md/render.gohtml":       markdownTemplate,
	"archive/archive.gohtml": archiveTemplate,
}

const themeCSS = `:root{--bg:#fff;--text:#333;--border:#ddd;--accent:#1967d2;--hover:#f5f5f5;--header-bg:#fafafa;--meta-bg:#f5f5f5;--meta-label:#555;--toc-bg:#fff;--toc-border:#ccc}
[data-theme=dark]{--bg:#1a1a2e;--text:#e0e0e0;--border:#333;--accent:#64b5f6;--hover:#2a2a3e;--header-bg:#16213e;--meta-bg:#2a2a3e;--meta-label:#aaa;--toc-bg:#1a1a2e;--toc-border:#444}
.topbar{display:flex;align-items:center;justify-content:space-between;padding:8px 16px;background:var(--header-bg);border-bottom:1px solid var(--border)}
.topbar .title{font-size:.95rem;font-weight:600}
.topbar .controls{display:flex;align-items:center;gap:8px}
.topbar button{background:none;border:1px solid var(--border);border-radius:4px;padding:4px 10px;color:var(--text);cursor:pointer;font-size:.8rem}
.topbar button:hover{background:var(--hover)}
.toc{position:relative}
.toc-drop{display:none;position:absolute;right:0;top:100%;background:var(--toc-bg);border:1px solid var(--toc-border);border-radius:4px;min-width:200px;max-height:300px;overflow-y:auto;z-index:10;margin-top:4px}
.toc.open .toc-drop{display:block}
.toc-drop a{display:block;padding:6px 12px;color:var(--accent);text-decoration:none;font-size:.85rem;border-bottom:1px solid var(--border)}
.toc-drop a:hover{background:var(--hover)}`

const themeJS = `(function(){var t=localStorage.getItem('blik-theme'),b=function(){var e=document.querySelector('.theme-btn');if(e)e.textContent=t==='dark'?'Light':'Dark'};if(!t){t=window.matchMedia('(prefers-color-scheme:dark)').matches?'dark':'light'}document.documentElement.setAttribute('data-theme',t);b()})();function toggleTheme(){var h=document.documentElement,t=h.getAttribute('data-theme')==='dark'?'light':'dark';h.setAttribute('data-theme',t);localStorage.setItem('blik-theme',t);var b=document.querySelector('.theme-btn');if(b)b.textContent=t==='dark'?'Light':'Dark'}`

const listingTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>{{.Title}}</title>
<style>
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;max-width:800px;margin:24px auto;padding:0 16px;background:var(--bg);color:var(--text)}
` + themeCSS + `
table{width:100%;border-collapse:collapse;margin-top:16px}
th,td{text-align:left;padding:4px 8px}
th{border-bottom:2px solid var(--border);font-weight:600;font-size:.85rem;color:var(--meta-label)}
td{border-bottom:1px solid var(--border)}
tr:hover td{background:var(--hover)}
a{color:var(--accent);text-decoration:none}
a:hover{text-decoration:underline}
.size{text-align:right;font-size:.85rem;color:var(--meta-label);white-space:nowrap}
.time{font-size:.85rem;color:var(--meta-label);white-space:nowrap}
.info{text-align:center}
.info a{padding:2px 8px;border:1px solid var(--border);border-radius:4px;font-size:.75rem;color:var(--meta-label);cursor:pointer}
.info a:hover{background:var(--hover);text-decoration:none}
</style>
</head>
<body>
<header class="topbar">
<span class="title">{{.Title}}</span>
<div class="controls">
<button class="theme-btn" onclick="toggleTheme()">Toggle</button>
</div>
</header>
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
<script>` + themeJS + `</script>
</body>
</html>`

const markdownTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>{{.Title}}</title>
<style>
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;max-width:800px;margin:24px auto;padding:0 16px;background:var(--bg);color:var(--text)}
.container{line-height:1.7;padding:16px 0}
.container img{max-width:100%}
.container pre{background:var(--hover);padding:12px;border-radius:4px;overflow-x:auto}
.container code{font-size:.9rem}
.container a{color:var(--accent)}
.container h2,.container h3,.container h4,.container h5,.container h6{margin-top:24px;color:var(--text)}
` + themeCSS + `
{{if .CSS}}{{.CSS}}{{end}}
</style>
<style media="(prefers-color-scheme:dark)">{{if .DarkCSS}}{{.DarkCSS}}{{end}}</style>
<style media="print">{{if .PrintCSS}}{{.PrintCSS}}{{end}}</style>
</head>
<body>
<header class="topbar">
<span class="title">{{.Title}}</span>
<div class="controls">
{{if .Headings}}<div class="toc">
<button onclick="this.parentElement.classList.toggle('open')">Contents</button>
<div class="toc-drop">
{{range .Headings}}<a href="#{{.Anchor}}" style="padding-left:{{.Pad}}px">{{.Text}}</a>
{{end}}</div>
</div>{{end}}
<button class="theme-btn" onclick="toggleTheme()">Toggle</button>
</div>
</header>
<div class="container">{{.Content}}</div>
<script>` + themeJS + `</script>
</body>
</html>`

const archiveTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>{{.FileName}}</title>
<style>
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;max-width:800px;margin:24px auto;padding:0 16px;background:var(--bg);color:var(--text)}
h1{font-size:1.2rem;font-weight:600}
.meta{display:flex;gap:24px;margin:16px 0;padding:12px;background:var(--meta-bg);border-radius:8px}
.meta-item{font-size:.85rem}
.meta-item strong{display:block;font-size:.75rem;color:var(--meta-label)}
details{margin:4px 0}
details summary{cursor:pointer;padding:2px 4px;border-radius:4px;font-size:.9rem}
details summary:hover{background:var(--hover)}
.file{display:flex;justify-content:space-between;padding:2px 4px 2px 20px;font-size:.85rem}
.file span{color:var(--meta-label)}
.dir{font-weight:500}
.dir::before{content:"\25b8 ";color:var(--meta-label)}
details[open]>.dir::before{content:"\25be "}
` + themeCSS + `
{{if .CSS}}{{.CSS}}{{end}}
</style>
<style media="(prefers-color-scheme:dark)">{{if .DarkCSS}}{{.DarkCSS}}{{end}}</style>
<style media="print">{{if .PrintCSS}}{{.PrintCSS}}{{end}}</style>
</head>
<body>
<header class="topbar">
<span class="title">{{.FileName}}</span>
<div class="controls">
<button class="theme-btn" onclick="toggleTheme()">Toggle</button>
</div>
</header>
<h1>{{.FileName}}</h1>
<div class="meta">
<div class="meta-item"><strong>Format</strong>{{.Format}}</div>
<div class="meta-item"><strong>Entries</strong>{{.FileCount}}</div>
<div class="meta-item"><strong>Size</strong>{{.Size}}</div>
</div>
{{.Tree}}
<script>` + themeJS + `</script>
</body>
</html>`
