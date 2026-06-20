package render

import (
	"testing"
)

func TestStripHTMLTags(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"<b>bold</b>", "bold"},
		{"<a href='x'>link</a>", "link"},
		{"no tags", "no tags"},
		{"", ""},
		{"<code>fmt.Println()</code>", "fmt.Println()"},
	}
	for _, tt := range tests {
		got := stripHTMLTags(tt.input)
		if got != tt.want {
			t.Errorf("stripHTMLTags(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestExtractHeadings(t *testing.T) {
	html := `<h1 id="introduction">Introduction</h1>
<p>Some text</p>
<h2 id="getting-started">Getting Started</h2>
<h3 id="installation">Installation</h3>
<h2 id="usage">Usage</h2>`

	headings := extractHeadings(html)
	if len(headings) != 4 {
		t.Fatalf("expected 4 headings, got %d", len(headings))
	}

	checks := []struct {
		level  int
		text   string
		anchor string
		pad    int
	}{
		{1, "Introduction", "introduction", 0},
		{2, "Getting Started", "getting-started", 20},
		{3, "Installation", "installation", 40},
		{2, "Usage", "usage", 20},
	}

	for i, c := range checks {
		h := headings[i]
		if h.Level != c.level || h.Text != c.text || h.Anchor != c.anchor || h.Pad != c.pad {
			t.Errorf("heading[%d] = {Level:%d, Text:%q, Anchor:%q, Pad:%d}, want {Level:%d, Text:%q, Anchor:%q, Pad:%d}",
				i, h.Level, h.Text, h.Anchor, h.Pad, c.level, c.text, c.anchor, c.pad)
		}
	}
}

func TestExtractHeadingsWithAttrs(t *testing.T) {
	html := `<h1 id="title" class="main">Title</h1><h2 id="sub" style="color:red">Sub</h2>`
	headings := extractHeadings(html)
	if len(headings) != 2 {
		t.Fatalf("expected 2 headings, got %d", len(headings))
	}
	if headings[0].Anchor != "title" || headings[0].Text != "Title" {
		t.Errorf("heading[0] = %+v", headings[0])
	}
	if headings[1].Anchor != "sub" || headings[1].Text != "Sub" {
		t.Errorf("heading[1] = %+v", headings[1])
	}
}

func TestExtractHeadingsNone(t *testing.T) {
	html := `<p>No headings here</p>`
	headings := extractHeadings(html)
	if headings != nil {
		t.Error("expected nil for no headings")
	}
}

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
