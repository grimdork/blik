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
	Thumbnails       bool
	Symlinks         bool
}

func defaultConfig() *Config {
	return &Config{
		Thumbnails: true,
		Symlinks:   true,
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
	if !hasBlik {
		local.Thumbnails = parent.Thumbnails
		local.Symlinks = parent.Symlinks
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
