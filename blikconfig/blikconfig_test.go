package blikconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSplitPatterns(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"*.md, *.html", []string{"*.md", "*.html"}},
		{"*.go", []string{"*.go"}},
		{"", nil},
		{" , ", nil},
		{"a,b,c", []string{"a", "b", "c"}},
	}
	for _, tt := range tests {
		got := splitPatterns(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("splitPatterns(%q) = %v, want %v", tt.input, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("splitPatterns(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
			}
		}
	}
}

func TestMergeConfigs(t *testing.T) {
	parent := &Config{
		MarkdownPatterns: []string{"*.md"},
		ArchivePatterns:  []string{"*.zip"},
		InfoPatterns:     []string{"*.zip", "*.tar.gz"},
		MarkdownTemplate: "custom/md",
		ArchiveTemplate:  "custom/archive",
	}

	t.Run("inherits from parent when local is empty", func(t *testing.T) {
		local := &Config{}
		got := mergeConfigs(parent, local, false)
		if got != local {
			t.Error("mergeConfigs should return local")
		}
		if len(got.MarkdownPatterns) != 1 || got.MarkdownPatterns[0] != "*.md" {
			t.Error("should inherit markdown patterns")
		}
		if len(got.ArchivePatterns) != 1 || got.ArchivePatterns[0] != "*.zip" {
			t.Error("should inherit archive patterns")
		}
		if got.MarkdownTemplate != "custom/md" {
			t.Error("should inherit markdown template")
		}
		if got.ArchiveTemplate != "custom/archive" {
			t.Error("should inherit archive template")
		}
	})

	t.Run("keeps local values when set", func(t *testing.T) {
		local := &Config{
			MarkdownPatterns: []string{"*.mdx"},
			ArchiveTemplate:  "local/archive",
		}
		got := mergeConfigs(parent, local, true)
		if len(got.MarkdownPatterns) != 1 || got.MarkdownPatterns[0] != "*.mdx" {
			t.Error("should keep local markdown patterns")
		}
		if got.ArchiveTemplate != "local/archive" {
			t.Error("should keep local archive template")
		}
		if len(got.ArchivePatterns) != 1 || got.ArchivePatterns[0] != "*.zip" {
			t.Error("should inherit archive patterns")
		}
	})
}

func TestMatchHandler(t *testing.T) {
	cfg := &Config{
		MarkdownPatterns: []string{"*.md", "*.markdown"},
		ArchivePatterns:  []string{"*.zip", "*.tar.gz", "*.tgz"},
	}

	tests := []struct {
		name string
		want string
	}{
		{"readme.md", "markdown"},
		{"article.markdown", "markdown"},
		{"archive.zip", "archive"},
		{"bundle.tar.gz", "archive"},
		{"file.tgz", "archive"},
		{"main.go", ""},
		{".blik", ""},
		{"file.txt", ""},
	}
	for _, tt := range tests {
		got := cfg.MatchHandler(tt.name)
		if got != tt.want {
			t.Errorf("MatchHandler(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestHasInfo(t *testing.T) {
	cfg := &Config{
		InfoPatterns: []string{"*.zip", "*.tar.gz", "*.tgz", "*.tar"},
	}

	tests := []struct {
		name string
		want bool
	}{
		{"file.zip", true},
		{"file.tar.gz", true},
		{"file.tgz", true},
		{"file.tar", true},
		{"readme.md", false},
		{"main.go", false},
	}
	for _, tt := range tests {
		got := cfg.HasInfo(tt.name)
		if got != tt.want {
			t.Errorf("HasInfo(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestLoadFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".blik")
	content := `[*.md, *.markdown]
type=markdown
template=blog/md

[*.zip, *.tar.gz]
type=archive

[*.jpg, *.png]
type=info
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := loadFile(path)
	if cfg == nil {
		t.Fatal("loadFile returned nil")
	}
	if len(cfg.MarkdownPatterns) != 2 {
		t.Errorf("expected 2 markdown patterns, got %d", len(cfg.MarkdownPatterns))
	}
	if len(cfg.ArchivePatterns) != 2 {
		t.Errorf("expected 2 archive patterns, got %d", len(cfg.ArchivePatterns))
	}
	if len(cfg.InfoPatterns) != 4 {
		t.Errorf("expected 4 info patterns (2 archive + 2 info), got %d", len(cfg.InfoPatterns))
	}
	if cfg.MarkdownTemplate != "blog/md" {
		t.Errorf("expected markdown_template 'blog/md', got %q", cfg.MarkdownTemplate)
	}
}

func TestLoadFileSections(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".blik")
	content := `[*.go]
type=markdown
template=code/md

[*.tar]
type=archive
template=store/archive

[*.md]
type=info
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := loadFile(path)
	if cfg == nil {
		t.Fatal("loadFile returned nil")
	}

	if len(cfg.MarkdownPatterns) != 1 || cfg.MarkdownPatterns[0] != "*.go" {
		t.Error("expected markdown pattern *.go")
	}
	if cfg.MarkdownTemplate != "code/md" {
		t.Errorf("expected markdown template 'code/md', got %q", cfg.MarkdownTemplate)
	}
	if len(cfg.ArchivePatterns) != 1 || cfg.ArchivePatterns[0] != "*.tar" {
		t.Error("expected archive pattern *.tar")
	}
	if cfg.ArchiveTemplate != "store/archive" {
		t.Errorf("expected archive template 'store/archive', got %q", cfg.ArchiveTemplate)
	}
	if len(cfg.InfoPatterns) != 2 {
		t.Errorf("expected 2 info patterns (1 archive + 1 info), got %d", len(cfg.InfoPatterns))
	}
}

func TestLoadFileMissing(t *testing.T) {
	cfg := loadFile("/nonexistent/.blik")
	if cfg != nil {
		t.Error("loadFile should return nil for missing file")
	}
}

func TestLoadFileInvalid(t *testing.T) {
	dir := t.TempDir()
	os.Remove(dir)
	cfg := loadFile(filepath.Join(dir, ".blik"))
	if cfg != nil {
		t.Error("loadFile should return nil when parent dir missing")
	}
}

func TestStoreGetConfig(t *testing.T) {
	root := t.TempDir()
	s := NewStore(root)

	cfg := s.GetConfig(root)
	if cfg == nil {
		t.Fatal("GetConfig returned nil")
	}

	s.Invalidate(root)
}

func TestStoreGetConfigCaches(t *testing.T) {
	root := t.TempDir()
	s := NewStore(root)

	c1 := s.GetConfig(root)
	c2 := s.GetConfig(root)
	if c1 != c2 {
		t.Error("GetConfig should return cached config")
	}

	s.Invalidate(root)
	c3 := s.GetConfig(root)
	if c3 == c1 {
		t.Error("GetConfig should return fresh config after Invalidate")
	}
}
