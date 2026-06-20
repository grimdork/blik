package template

import (
	"strings"
	"testing"
)

func TestRenderBuiltin(t *testing.T) {
	e := NewEngine("")
	out, err := e.Render("webroot/listing.gohtml", map[string]any{
		"Title":   "Index of /",
		"Entries": []dirEntry{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Index of /") {
		t.Error("output should contain title")
	}
	if !strings.Contains(out, "<table>") {
		t.Error("output should contain table")
	}
}

func TestCSSBuiltin(t *testing.T) {
	e := NewEngine("")
	css, darkCSS, printCSS := e.CSS("webroot")
	if css == "" {
		t.Error("expected non-empty CSS")
	}
	if darkCSS == "" {
		t.Error("expected non-empty dark CSS")
	}
	if printCSS == "" {
		t.Error("expected non-empty print CSS")
	}
	if !strings.Contains(css, "--bg") {
		t.Error("CSS should contain theme variables")
	}
}

type dirEntry struct {
	Name    string
	Size    string
	ModTime string
	IsDir   bool
	HasInfo bool
}
