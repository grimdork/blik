package blikconfig

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/grimdork/climate/ini"
)

type Config struct {
	MarkdownPatterns []string
	ArchivePatterns  []string
	InfoPatterns     []string
	MarkdownTemplate string
	ArchiveTemplate  string
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

func (s *Store) loadConfig(dir string) *Config {
	cfg := loadFile(filepath.Join(dir, ".blik"))
	if cfg == nil {
		cfg = &Config{}
	}

	if dir != s.root {
		parent := filepath.Dir(dir)
		// Stop at filesystem root (parent == dir) and above our serve root.
		if parent != dir && strings.HasPrefix(dir, s.root) {
			parentCfg := s.GetConfig(parent)
			cfg = mergeConfigs(parentCfg, cfg)
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

	cfg := &Config{}
	if v := inif.GetString("", "markdown_patterns"); v != "" {
		cfg.MarkdownPatterns = splitPatterns(v)
	}
	if v := inif.GetString("", "archive_patterns"); v != "" {
		cfg.ArchivePatterns = splitPatterns(v)
	}
	if v := inif.GetString("", "info_patterns"); v != "" {
		cfg.InfoPatterns = splitPatterns(v)
	}
	cfg.MarkdownTemplate = inif.GetString("", "markdown_template")
	cfg.ArchiveTemplate = inif.GetString("", "archive_template")
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

func mergeConfigs(parent, local *Config) *Config {
	cfg := &Config{}
	if len(local.MarkdownPatterns) > 0 {
		cfg.MarkdownPatterns = local.MarkdownPatterns
	} else {
		cfg.MarkdownPatterns = parent.MarkdownPatterns
	}
	if len(local.ArchivePatterns) > 0 {
		cfg.ArchivePatterns = local.ArchivePatterns
	} else {
		cfg.ArchivePatterns = parent.ArchivePatterns
	}
	if len(local.InfoPatterns) > 0 {
		cfg.InfoPatterns = local.InfoPatterns
	} else {
		cfg.InfoPatterns = parent.InfoPatterns
	}
	if local.MarkdownTemplate != "" {
		cfg.MarkdownTemplate = local.MarkdownTemplate
	} else {
		cfg.MarkdownTemplate = parent.MarkdownTemplate
	}
	if local.ArchiveTemplate != "" {
		cfg.ArchiveTemplate = local.ArchiveTemplate
	} else {
		cfg.ArchiveTemplate = parent.ArchiveTemplate
	}
	return cfg
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
