package render

import (
	"regexp"

	"github.com/grimdork/markulator"
)

type Heading struct {
	Level  int
	Text   string
	Anchor string
	Pad    int
}

type Result struct {
	HTML     string
	Headings []Heading
}

var (
	headingRe = regexp.MustCompile(`<h([1-6])(?:\s+[^>]*?)?\s+id="([^"]+)"[^>]*>(.*?)</h[1-6]>`)
	stripRe   = regexp.MustCompile(`<[^>]*>`)
)

func Markdown(src []byte) (*Result, error) {
	r := mk.NewRenderer(
		mk.WithSingleFile(true),
		mk.WithDoctype(false),
		mk.WithBodyTags(false),
	)
	v, err := r.RenderString(src)
	if err != nil {
		return nil, err
	}

	html := string(v)
	headings := extractHeadings(html)

	return &Result{
		HTML:     html,
		Headings: headings,
	}, nil
}

func extractHeadings(html string) []Heading {
	matches := headingRe.FindAllStringSubmatch(html, -1)
	if len(matches) == 0 {
		return nil
	}
	headings := make([]Heading, 0, len(matches))
	for _, m := range matches {
		level := int(m[1][0] - '0')
		headings = append(headings, Heading{
			Level:  level,
			Text:   stripHTMLTags(m[3]),
			Anchor: m[2],
			Pad:    (level - 1) * 20,
		})
	}
	return headings
}

func stripHTMLTags(s string) string {
	return stripRe.ReplaceAllString(s, "")
}

