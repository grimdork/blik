package render

import (
	"testing"
)

func TestMarkdown(t *testing.T) {
	src := []byte("# Hello\n\nThis is a paragraph.\n\n## Section One\n\nContent here.")
	result, err := Markdown(src)
	if err != nil {
		t.Fatal(err)
	}

	if result.HTML == "" {
		t.Error("expected non-empty HTML")
	}

	if len(result.Headings) != 2 {
		t.Fatalf("expected 2 headings, got %d", len(result.Headings))
	}

	if result.Headings[0].Level != 1 {
		t.Errorf("expected h1, got h%d", result.Headings[0].Level)
	}
	if result.Headings[1].Level != 2 {
		t.Errorf("expected h2, got h%d", result.Headings[1].Level)
	}
}

func TestMarkdownCodeHighlighting(t *testing.T) {
	src := []byte("```go\npackage main\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n```")
	result, err := Markdown(src)
	if err != nil {
		t.Fatal(err)
	}

	html := result.HTML
	if len(html) == 0 {
		t.Fatal("expected non-empty HTML")
	}
}
