package render

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
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

type hlPattern struct {
	re    *regexp.Regexp
	class string
}

var hlPatterns = []hlPattern{
	{regexp.MustCompile(`"(?:[^"\\]|\\.)*"`), "hl-string"},
	{regexp.MustCompile(`'(?:[^'\\]|\\.)*'`), "hl-string"},
	{regexp.MustCompile("`[^`]*`"), "hl-string"},
	{regexp.MustCompile(`//[^\n]*`), "hl-comment"},
	{regexp.MustCompile(`#[^\n]*`), "hl-comment"},
	{regexp.MustCompile(`;[^\n]*`), "hl-comment"},
	{regexp.MustCompile(`\b[0-9]+(?:\.[0-9]+)?\b`), "hl-number"},
	{regexp.MustCompile(`\b(?:bool|break|byte|case|chan|complex64|complex128|const|continue|default|defer|do|done|elif|else|esac|error|fallthrough|fi|float32|float64|for|func|function|go|goto|if|import|in|int|int16|int32|int64|int8|interface|map|package|range|return|rune|select|string|struct|switch|then|time|type|uint|uint16|uint32|uint64|uint8|uintptr|until|var|while)\b`), "hl-keyword"},
	{regexp.MustCompile(`\b([a-zA-Z_][a-zA-Z0-9_]*)\(`), "hl-func"},
}

var hlRe = buildHLRe()

func buildHLRe() *regexp.Regexp {
	var parts []string
	for _, p := range hlPatterns {
		parts = append(parts, "("+p.re.String()+")")
	}
	return regexp.MustCompile(strings.Join(parts, "|"))
}

type customRenderer struct{}

func (r *customRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindHeading, r.renderHeading)
	reg.Register(ast.KindFencedCodeBlock, r.renderFencedCodeBlock)
}

func (r *customRenderer) renderHeading(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.Heading)
	if entering {
		_, _ = w.WriteString("<h")
		_ = w.WriteByte("0123456"[n.Level])
		if n.Attributes() != nil {
			html.RenderAttributes(w, node, html.HeadingAttributeFilter)
		}
		_ = w.WriteByte('>')
	} else {
		_, _ = w.WriteString("</h")
		_ = w.WriteByte("0123456"[n.Level])
		_, _ = w.WriteString(">\n")
	}
	return ast.WalkContinue, nil
}

func (r *customRenderer) renderFencedCodeBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.FencedCodeBlock)
	if !entering {
		return ast.WalkContinue, nil
	}
	lang := "text"
	if info := n.Language(source); info != nil {
		lang = string(info)
	}
	var buf bytes.Buffer
	for i := 0; i < n.Lines().Len(); i++ {
		line := n.Lines().At(i)
		buf.Write(line.Value(source))
	}
	highlighted := highlightCode(buf.String(), lang)
	_, _ = w.WriteString("<pre><code")
	if lang != "text" {
		_, _ = w.WriteString(` class="language-`)
		_, _ = w.WriteString(lang)
		_, _ = w.WriteString(`"`)
	}
	_, _ = w.WriteString(">")
	_, _ = w.WriteString(highlighted)
	_, _ = w.WriteString("</code></pre>\n")
	return ast.WalkSkipChildren, nil
}

func Markdown(src []byte) (*Result, error) {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.Table,
			extension.Strikethrough,
			extension.TaskList,
			extension.DefinitionList,
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
			renderer.WithNodeRenderers(
				util.Prioritized(&customRenderer{}, 200),
			),
		),
	)
	reader := text.NewReader(src)
	doc := md.Parser().Parse(reader)
	headings := walkPreprocess(doc, src)
	var buf strings.Builder
	if err := md.Renderer().Render(&buf, src, doc); err != nil {
		return nil, err
	}
	return &Result{
		HTML:     buf.String(),
		Headings: headings,
	}, nil
}

func walkPreprocess(doc ast.Node, source []byte) []Heading {
	var headings []Heading
	counts := map[string]int{}
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n.Kind() {
		case ast.KindHeading:
			heading := n.(*ast.Heading)
			text := collectText(heading, source)
			base := slugify(text)
			if base == "" {
				base = "heading"
			}
			counts[base]++
			id := base
			if counts[base] > 1 {
				id = fmt.Sprintf("%s-%d", base, counts[base])
			}
			n.SetAttribute([]byte("id"), []byte(id))
			n.SetAttribute([]byte("class"), []byte(fmt.Sprintf("heading-h%d", heading.Level)))
			headings = append(headings, Heading{
				Level:  heading.Level,
				Text:   text,
				Anchor: id,
				Pad:    (heading.Level - 1) * 20,
			})
		}
		return ast.WalkContinue, nil
	})
	return headings
}

func collectText(n ast.Node, source []byte) string {
	var buf strings.Builder
	ast.Walk(n, func(child ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if t, ok := child.(*ast.Text); ok {
			buf.Write(t.Value(source))
		}
		return ast.WalkContinue, nil
	})
	return buf.String()
}

func highlightCode(code string, lang string) string {
	s := hlRe.ReplaceAllStringFunc(code, func(match string) string {
		subs := hlRe.FindStringSubmatch(match)
		for i, p := range hlPatterns {
			if subs[i+1] != "" {
				if p.class == "hl-func" {
					return `<span class="hl-func">` + match[:len(match)-1] + `</span>(`
				}
				return `<span class="` + p.class + `">` + match + `</span>`
			}
		}
		return match
	})
	s = escapeOutsideSpans(s)
	if isShellLang(lang) {
		s = highlightShellFirstWord(s)
	}
	return s
}

var shellLangs = map[string]bool{
	"text":  true,
	"shell": true,
	"sh":    true,
	"bash":  true,
	"zsh":   true,
	"ksh":   true,
}

func isShellLang(lang string) bool {
	return shellLangs[lang]
}

func highlightShellFirstWord(s string) string {
	var buf strings.Builder
	for i, line := range strings.Split(s, "\n") {
		if i > 0 {
			buf.WriteByte('\n')
		}
		trimmed := strings.TrimLeft(line, " \t")
		if trimmed == "" || strings.HasPrefix(trimmed, "<span ") {
			buf.WriteString(line)
			continue
		}
		re := regexp.MustCompile(`^(\s*)([a-zA-Z_][a-zA-Z0-9_]*)`)
		m := re.FindStringSubmatch(line)
		if m != nil {
			buf.WriteString(m[1])
			buf.WriteString(`<span class="hl-command">`)
			buf.WriteString(m[2])
			buf.WriteString(`</span>`)
			buf.WriteString(line[len(m[0]):])
		} else {
			buf.WriteString(line)
		}
	}
	return buf.String()
}

func escapeOutsideSpans(s string) string {
	var buf strings.Builder
	depth := 0
	for i := 0; i < len(s); i++ {
		if strings.HasPrefix(s[i:], "<span ") {
			j := strings.IndexByte(s[i:], '>')
			if j >= 0 {
				buf.WriteString(s[i : i+j+1])
				i += j
				depth++
				continue
			}
		}
		if strings.HasPrefix(s[i:], "</span>") {
			buf.WriteString("</span>")
			i += 6
			depth--
			continue
		}
		if depth > 0 {
			buf.WriteByte(s[i])
		} else {
			switch s[i] {
			case '&':
				buf.WriteString("&amp;")
			case '<':
				buf.WriteString("&lt;")
			case '>':
				buf.WriteString("&gt;")
			default:
				buf.WriteByte(s[i])
			}
		}
	}
	return buf.String()
}

var nonAlnumRe = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	s = strings.ToLower(s)
	s = nonAlnumRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}
