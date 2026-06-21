package blikconfig

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/grimdork/climate/fx"
	"github.com/grimdork/climate/ini"
)

type Config struct {
	MarkdownPatterns []string
	ArchivePatterns  []string
	InfoPatterns     []string
	MarkdownTemplate string
	ArchiveTemplate  string
	DataPatterns     map[string][]string
	Thumbnails       bool
	Symlinks         bool
	IndexFiles       []string
	GenerateThumbs   bool
	ThumbSize        string
	ThumbWorkers     int
	Layout           string
	MdLayout         string
	CSP              string
	HSTS             string
}

func defaultConfig() *Config {
	return &Config{
		Thumbnails:     true,
		Symlinks:       true,
		GenerateThumbs: false,
		ThumbSize:      "256w",
		ThumbWorkers:   1,
		Layout:         "single",
		MdLayout:       "single",
		CSP:            "default-src 'none'; img-src 'self' data: https:; media-src 'self'; font-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'; connect-src 'self'; manifest-src 'self'",
		HSTS:           "max-age=31536000; includeSubDomains",
	}
}

type Store struct {
	mu    sync.RWMutex
	root  string
	cache map[string]*Config
}

func NewStore(root string) *Store {
	return &Store{
		root:  root,
		cache: make(map[string]*Config),
	}
}

func (s *Store) GetConfig(dir string) *Config {
	s.mu.RLock()
	if cfg, ok := s.cache[dir]; ok {
		s.mu.RUnlock()
		return cfg
	}
	s.mu.RUnlock()
	return s.loadConfig(dir)
}

func (s *Store) Invalidate(dir string) {
	s.mu.Lock()
	delete(s.cache, dir)
	s.mu.Unlock()
}

func (s *Store) Preload() {
	err := filepath.Walk(s.root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fx.Fprintln(os.Stderr, "{logstamp} {danger}preload:{@} {}: {}", path, err)
			return nil
		}
		if info.IsDir() {
			s.GetConfig(path)
		}
		return nil
	})
	if err != nil {
		fx.Fprintln(os.Stderr, "{logstamp} {danger}preload:{@} {}", err)
	}
}

func (s *Store) loadConfig(dir string) *Config {
	blikPath := filepath.Join(dir, ".blik")
	_, err := os.Stat(blikPath)
	hasBlik := err == nil

	cfg := loadFile(blikPath)
	if cfg == nil {
		cfg = defaultConfig()
	}

	if dir != s.root {
		parent := filepath.Dir(dir)
		// Stop at filesystem root (parent == dir) and above our serve root.
		if parent != dir && strings.HasPrefix(dir, s.root) {
			parentCfg := s.GetConfig(parent)
			cfg = mergeConfigs(parentCfg, cfg, hasBlik)
		}
	}

	s.mu.Lock()
	s.cache[dir] = cfg
	s.mu.Unlock()
	return cfg
}

func loadFile(path string) *Config {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	f.Close()

	inif, err := ini.Load(path)
	if err != nil {
		return nil
	}

	cfg := defaultConfig()
	loaded := 0
	for _, secName := range inif.Order {
		sec := inif.Sections[secName]
		if sec == nil {
			continue
		}

		if secName == "blik" {
			cfg.Thumbnails = sec.GetBool("thumbnails", cfg.Thumbnails)
			cfg.Symlinks = sec.GetBool("symlinks", cfg.Symlinks)
			if idx := sec.GetString("index", ""); idx != "" {
				cfg.IndexFiles = splitPatterns(idx)
			}
			cfg.GenerateThumbs = sec.GetBool("thumbs", cfg.GenerateThumbs)
			if sz := sec.GetString("thumbsize", ""); sz != "" {
				cfg.ThumbSize = sz
			}
			cfg.ThumbWorkers = int(sec.GetInt("workers", int64(cfg.ThumbWorkers)))
			if lay := sec.GetString("layout", ""); lay != "" {
				cfg.Layout = lay
			}
			if lay := sec.GetString("mdlayout", ""); lay != "" {
				cfg.MdLayout = lay
			}
			if v := sec.GetString("csp", ""); v != "" {
				cfg.CSP = v
			}
			if v := sec.GetString("hsts", ""); v != "" {
				cfg.HSTS = v
			}
			continue
		}

		t := sec.GetString("type", "")
		if t == "" {
			continue
		}
		loaded++

		patterns := splitPatterns(secName)
		switch t {
		case "markdown":
			cfg.MarkdownPatterns = append(cfg.MarkdownPatterns, patterns...)
			if tmpl := sec.GetString("template", ""); tmpl != "" {
				cfg.MarkdownTemplate = tmpl
			}
		case "archive":
			cfg.ArchivePatterns = append(cfg.ArchivePatterns, patterns...)
			cfg.InfoPatterns = append(cfg.InfoPatterns, patterns...)
			if tmpl := sec.GetString("template", ""); tmpl != "" {
				cfg.ArchiveTemplate = tmpl
			}
		case "info":
			cfg.InfoPatterns = append(cfg.InfoPatterns, patterns...)
		case "json", "xml", "yaml", "toml", "ini", "csv", "tsv":
			if cfg.DataPatterns == nil {
				cfg.DataPatterns = make(map[string][]string)
			}
			cfg.DataPatterns[t] = append(cfg.DataPatterns[t], patterns...)
		}
	}

	fx.Println("{logstamp} {info}config:{@} {} — loaded .blik with {} type(s)", path, loaded)
	return cfg
}

func splitPatterns(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func mergeConfigs(parent, local *Config, hasBlik bool) *Config {
	if len(local.MarkdownPatterns) == 0 {
		local.MarkdownPatterns = parent.MarkdownPatterns
	}
	if len(local.ArchivePatterns) == 0 {
		local.ArchivePatterns = parent.ArchivePatterns
	}
	if len(local.InfoPatterns) == 0 {
		local.InfoPatterns = parent.InfoPatterns
	}
	if local.MarkdownTemplate == "" {
		local.MarkdownTemplate = parent.MarkdownTemplate
	}
	if local.ArchiveTemplate == "" {
		local.ArchiveTemplate = parent.ArchiveTemplate
	}
	if len(local.IndexFiles) == 0 {
		local.IndexFiles = parent.IndexFiles
	}
	if !hasBlik {
		local.Thumbnails = parent.Thumbnails
		local.Symlinks = parent.Symlinks
		local.GenerateThumbs = parent.GenerateThumbs
		local.ThumbSize = parent.ThumbSize
		local.ThumbWorkers = parent.ThumbWorkers
		local.Layout = parent.Layout
		local.MdLayout = parent.MdLayout
		local.CSP = parent.CSP
		local.HSTS = parent.HSTS
		if local.DataPatterns == nil {
			local.DataPatterns = parent.DataPatterns
		} else {
			for format, patterns := range parent.DataPatterns {
				if _, ok := local.DataPatterns[format]; !ok {
					local.DataPatterns[format] = patterns
				}
			}
		}
	}
	return local
}

func (c *Config) MatchHandler(name string) string {
	for _, p := range c.MarkdownPatterns {
		if ok, _ := filepath.Match(p, name); ok {
			return "markdown"
		}
	}
	for _, p := range c.ArchivePatterns {
		if ok, _ := filepath.Match(p, name); ok {
			return "archive"
		}
	}
	for format, patterns := range c.DataPatterns {
		for _, p := range patterns {
			if ok, _ := filepath.Match(p, name); ok {
				return format
			}
		}
	}
	return ""
}

func (c *Config) HasInfo(name string) bool {
	for _, p := range c.InfoPatterns {
		if ok, _ := filepath.Match(p, name); ok {
			return true
		}
	}
	return false
}
