package content

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIconName(t *testing.T) {
	tests := []struct {
		name  string
		isDir bool
		want  string
	}{
		{"", true, "folder"},
		{"dir", true, "folder"},
		{"file.go", false, "file-code"},
		{"file.js", false, "file-type-js"},
		{"file.ts", false, "file-type-ts"},
		{"file.tsx", false, "file-type-tsx"},
		{"file.jsx", false, "file-type-jsx"},
		{"file.css", false, "file-type-css"},
		{"file.html", false, "file-type-html"},
		{"file.rs", false, "file-type-rs"},
		{"file.py", false, "file-code"},
		{"file.md", false, "file-text"},
		{"file.txt", false, "file-text"},
		{"file.jpg", false, "file-type-jpg"},
		{"file.jpeg", false, "file-type-jpg"},
		{"file.png", false, "file-type-png"},
		{"file.zip", false, "file-zip"},
		{"file.tar.gz", false, "file-zip"},
		{"file.mp3", false, "music"},
		{"file.mp4", false, "video"},
		{"file.pdf", false, "file-type-pdf"},
		{"file.unknown", false, "file-unknown"},
	}
	for _, tt := range tests {
		got := iconName(tt.name, tt.isDir)
		if got != tt.want {
			t.Errorf("iconName(%q, %v) = %q, want %q", tt.name, tt.isDir, got, tt.want)
		}
	}
}

func TestIsImage(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"photo.jpg", true},
		{"photo.jpeg", true},
		{"photo.png", true},
		{"photo.gif", true},
		{"photo.webp", true},
		{"photo.bmp", true},
		{"photo.ico", true},
		{"file.txt", false},
		{"file.go", false},
	}
	for _, tt := range tests {
		got := isImage(tt.name)
		if got != tt.want {
			t.Errorf("isImage(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestIconSVG(t *testing.T) {
	home, _ := os.UserHomeDir()
	iconDir := filepath.Join(home, "src", "MIT-free-icons", "icons")
	initIconCache(iconDir)

	svg := iconSVG("main.go", false)
	if svg == "" || strings.HasPrefix(svg, "<!--") {
		t.Errorf("iconSVG for .go returned empty or missing: %q", svg[:min(len(svg), 50)])
	}
	if !strings.Contains(svg, "<svg") {
		t.Errorf("iconSVG does not contain <svg tag")
	}

	svgDir := iconSVG("somedir", true)
	if !strings.Contains(svgDir, "<svg") {
		t.Errorf("iconSVG for dir does not contain <svg tag")
	}

	svgUnknown := iconSVG("file.xyz123", false)
	if !strings.Contains(svgUnknown, "<svg") {
		t.Errorf("iconSVG for unknown ext does not contain <svg tag")
	}
}

func TestGuessFormat(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"photo.jpg", "JPEG"},
		{"photo.jpeg", "JPEG"},
		{"photo.png", "PNG"},
		{"photo.gif", "GIF"},
		{"photo.webp", "WebP"},
		{"photo.bmp", "BMP"},
		{"photo.ico", "ICO"},
		{"file.txt", ".txt"},
	}
	for _, tt := range tests {
		got := guessFormat(tt.name)
		if got != tt.want {
			t.Errorf("guessFormat(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0 B"},
		{1, "1 B"},
		{1023, "1023 B"},
		{1024, "1.0 KiB"},
		{1536, "1.5 KiB"},
		{1048576, "1.0 MiB"},
		{1073741824, "1.0 GiB"},
		{1610612736, "1.5 GiB"},
	}
	for _, tt := range tests {
		got := formatSize(tt.input)
		if got != tt.want {
			t.Errorf("formatSize(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
