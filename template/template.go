package template

import (
	"embed"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

//go:embed tpl/webroot/* tpl/md/* tpl/archive/* tpl/data/*
var templateFS embed.FS

type Engine struct {
	dir   string
	mu    sync.RWMutex
	cache map[string]*template.Template
	sri   string
}

func NewEngine(dir string) *Engine {
	return &Engine{
		dir:   dir,
		cache: make(map[string]*template.Template),
	}
}

func (e *Engine) SetSRI(sri string) {
	e.sri = sri
}

func (e *Engine) Render(name string, data any) (string, error) {
	tmpl, err := e.load(name)
	if err != nil {
		return "", err
	}
	if e.sri != "" {
		if m, ok := data.(map[string]any); ok {
			m["SRI"] = e.sri
		}
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
	if e.dir != "" {
		p := filepath.Join(e.dir, path)
		b, err := os.ReadFile(p)
		if err == nil {
			return string(b)
		}
	}
	b, err := templateFS.ReadFile("tpl/" + path)
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

	b, err := templateFS.ReadFile("tpl/" + name)
	if err != nil {
		return nil, os.ErrNotExist
	}
	t, err = t.Parse(string(b))
	if err != nil {
		return nil, err
	}
	e.mu.Lock()
	e.cache[name] = t
	e.mu.Unlock()
	return t, nil
}
