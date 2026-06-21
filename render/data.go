package render

import (
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/grimdork/climate/ini"
	"gopkg.in/yaml.v3"
)

type Node struct {
	Key      string
	Value    string
	Type     string
	Children []Node
}

type Table struct {
	Headers  []string
	Rows     [][]string
	RowCount int
}

func guessType(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "null"
	}
	if s == "true" || s == "false" {
		return "bool"
	}
	if _, err := strconv.ParseFloat(s, 64); err == nil {
		return "number"
	}
	return "string"
}

func TreeFromJSON(r io.Reader) (Node, error) {
	var v any
	if err := json.NewDecoder(r).Decode(&v); err != nil {
		return Node{}, err
	}
	return nodeFromAny("", v), nil
}

func nodeFromAny(key string, v any) Node {
	switch val := v.(type) {
	case map[string]any:
		n := Node{Key: key, Type: "object"}
		for k, child := range val {
			n.Children = append(n.Children, nodeFromAny(k, child))
		}
		return n
	case []any:
		n := Node{Key: key, Type: "array"}
		for i, child := range val {
			n.Children = append(n.Children, nodeFromAny(fmt.Sprintf("%d", i), child))
		}
		return n
	case string:
		return Node{Key: key, Value: val, Type: guessType(val)}
	case float64:
		return Node{Key: key, Value: strconv.FormatFloat(val, 'f', -1, 64), Type: "number"}
	case bool:
		return Node{Key: key, Value: strconv.FormatBool(val), Type: "bool"}
	case nil:
		return Node{Key: key, Type: "null"}
	default:
		return Node{Key: key, Value: fmt.Sprintf("%v", val), Type: "string"}
	}
}

func TreeFromYAML(r io.Reader) (Node, error) {
	var v any
	decoder := yaml.NewDecoder(r)
	if err := decoder.Decode(&v); err != nil {
		return Node{}, err
	}
	return nodeFromAny("", v), nil
}

func TreeFromTOML(r io.Reader) (Node, error) {
	var v map[string]any
	if _, err := toml.NewDecoder(r).Decode(&v); err != nil {
		return Node{}, err
	}
	return nodeFromAny("", v), nil
}

func TreeFromINI(r io.Reader) (Node, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return Node{}, err
	}

	tmp, err := os.CreateTemp("", "blik-*.ini")
	if err != nil {
		return Node{}, err
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return Node{}, err
	}
	tmp.Close()

	f, err := ini.Load(tmp.Name())
	if err != nil {
		return Node{}, err
	}

	root := Node{Key: "", Type: "object"}

	// Top-level properties
	for _, key := range f.PropOrder {
		fields := f.Properties[key]
		if len(fields) == 0 {
			continue
		}
		root.Children = append(root.Children, Node{
			Key:   key,
			Value: fields[0].Value,
			Type:  guessType(fields[0].Value),
		})
	}

	// Section properties
	for _, secName := range f.Order {
		sec := f.Sections[secName]
		if sec == nil {
			continue
		}
		secNode := Node{Key: secName, Type: "object"}
		for _, key := range sec.Order {
			sectionFields := sec.Fields[key]
			if len(sectionFields) == 0 {
				continue
			}
			secNode.Children = append(secNode.Children, Node{
				Key:   key,
				Value: sectionFields[0].Value,
				Type:  guessType(sectionFields[0].Value),
			})
		}
		root.Children = append(root.Children, secNode)
	}

	return root, nil
}

func TreeFromXML(r io.Reader) (Node, error) {
	decoder := xml.NewDecoder(r)
	root := Node{Type: "object"}
	stack := []*Node{&root}
	var textBuf strings.Builder

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return Node{}, err
		}
		switch t := token.(type) {
		case xml.StartElement:
			textBuf.Reset()
			child := Node{Key: t.Name.Local, Type: "object"}
			for _, attr := range t.Attr {
				child.Children = append(child.Children, Node{
					Key:   "@" + attr.Name.Local,
					Value: attr.Value,
					Type:  guessType(attr.Value),
				})
			}
			parent := stack[len(stack)-1]
			parent.Children = append(parent.Children, child)
			stack = append(stack, &parent.Children[len(parent.Children)-1])

		case xml.CharData:
			textBuf.Write(t)

		case xml.EndElement:
			node := stack[len(stack)-1]
			text := strings.TrimSpace(textBuf.String())
			if text != "" {
				node.Value = text
				node.Type = guessType(text)
			}
			if len(node.Children) == 0 && text == "" {
				node.Type = "null"
			}
			stack = stack[:len(stack)-1]
		}
	}
	return root, nil
}

func TableFromCSV(r io.Reader, comma rune) (Table, error) {
	return parseDelimited(r, comma)
}

func parseDelimited(r io.Reader, comma rune) (Table, error) {
	reader := csv.NewReader(r)
	reader.Comma = comma
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1

	all, err := reader.ReadAll()
	if err != nil {
		return Table{}, err
	}
	if len(all) == 0 {
		return Table{}, nil
	}

	t := Table{
		Headers:  all[0],
		RowCount: len(all) - 1,
	}
	if len(all) > 1 {
		t.Rows = all[1:]
	}
	return t, nil
}

func RenderTree(root Node) string {
	var buf strings.Builder
	renderNode(&buf, root, 0)
	return buf.String()
}

func renderNode(buf *strings.Builder, n Node, depth int) {
	if len(n.Children) == 0 {
		buf.WriteString(`<div class="leaf">`)
		if n.Key != "" {
			buf.WriteString(`<span class="key">`)
			writeEscaped(buf, n.Key)
			buf.WriteString(`</span>`)
			buf.WriteString(`<span class="sep">: </span>`)
		}
		valClass := "val-" + n.Type
		if n.Type == "null" {
			buf.WriteString(`<span class="` + valClass + `">null</span>`)
		} else {
			buf.WriteString(`<span class="` + valClass + `">`)
			writeEscaped(buf, n.Value)
			buf.WriteString(`</span>`)
		}
		if n.Type != "" && n.Type != "string" {
			buf.WriteString(`<span class="type-tag">` + n.Type + `</span>`)
		}
		buf.WriteString("</div>\n")
		return
	}

	open := ""
	if depth < 2 {
		open = " open"
	}
	buf.WriteString(`<details` + open + `>`)
	buf.WriteString(`<summary>`)
	writeEscaped(buf, n.Key)
	if n.Type == "array" {
		buf.WriteString(fmt.Sprintf(` <span class="type-tag">[%d]</span>`, len(n.Children)))
	}
	buf.WriteString("</summary>\n")
	for _, child := range n.Children {
		renderNode(buf, child, depth+1)
	}
	buf.WriteString("</details>\n")
}

func writeEscaped(buf *strings.Builder, s string) {
	for _, r := range s {
		switch r {
		case '<':
			buf.WriteString("&lt;")
		case '>':
			buf.WriteString("&gt;")
		case '&':
			buf.WriteString("&amp;")
		case '"':
			buf.WriteString("&quot;")
		case '\'':
			buf.WriteString("&#39;")
		default:
			buf.WriteRune(r)
		}
	}
}
