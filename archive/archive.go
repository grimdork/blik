package archive

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"html"
	"io"
	"os"
	"sort"
	"strings"
)

type Entry struct {
	Name  string
	Size  int64
	IsDir bool
}

type FileInfo struct {
	Format    string
	FileCount int
	TotalSize int64
	Entries   []Entry
	TreeHTML  string
}

func Read(path string) (*FileInfo, error) {
	switch {
	case strings.HasSuffix(path, ".zip"):
		return readZip(path)
	case strings.HasSuffix(path, ".tar.gz") || strings.HasSuffix(path, ".tgz"):
		return readTarGz(path)
	case strings.HasSuffix(path, ".tar"):
		return readTar(path)
	default:
		return nil, fmt.Errorf("unsupported archive format: %s", path)
	}
}

func readZip(path string) (*FileInfo, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	info := &FileInfo{Format: "ZIP"}
	tree := newTreeBuilder()
	for _, f := range r.File {
		mode := f.Mode()
		e := Entry{
			Name:  f.Name,
			Size:  int64(f.UncompressedSize64),
			IsDir: mode.IsDir(),
		}
		if !e.IsDir {
			info.FileCount++
			info.TotalSize += e.Size
		}
		info.Entries = append(info.Entries, e)
		tree.add(e.Name, e.IsDir)
	}
	sortEntries(info.Entries)
	info.TreeHTML = tree.build()
	return info, nil
}

func readTar(path string) (*FileInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return readTarReader(f, "tar")
}

func readTarGz(path string) (*FileInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gzr.Close()
	return readTarReader(gzr, "tar.gz")
}

func readTarReader(r io.Reader, format string) (*FileInfo, error) {
	tr := tar.NewReader(r)
	info := &FileInfo{Format: format}
	tree := newTreeBuilder()
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		e := Entry{
			Name:  hdr.Name,
			Size:  hdr.Size,
			IsDir: hdr.FileInfo().IsDir(),
		}
		if !e.IsDir {
			info.FileCount++
			info.TotalSize += e.Size
		}
		info.Entries = append(info.Entries, e)
		tree.add(e.Name, e.IsDir)
	}
	sortEntries(info.Entries)
	info.TreeHTML = tree.build()
	return info, nil
}

func sortEntries(entries []Entry) {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return entries[i].Name < entries[j].Name
	})
}

type treeNode struct {
	name     string
	children map[string]*treeNode
	isDir    bool
}

type treeBuilder struct {
	root *treeNode
}

func newTreeBuilder() *treeBuilder {
	return &treeBuilder{
		root: &treeNode{children: make(map[string]*treeNode)},
	}
}

func (tb *treeBuilder) add(path string, isDir bool) {
	parts := strings.Split(path, "/")
	node := tb.root
	for _, part := range parts {
		if part == "" {
			continue
		}
		child, ok := node.children[part]
		if !ok {
			child = &treeNode{name: part, children: make(map[string]*treeNode)}
			node.children[part] = child
		}
		node = child
	}
	node.isDir = isDir
}

func (tb *treeBuilder) build() string {
	var b strings.Builder
	tb.buildNode(&b, tb.root, 0)
	return b.String()
}

func (tb *treeBuilder) buildNode(b *strings.Builder, node *treeNode, depth int) {
	names := make([]string, 0, len(node.children))
	for name := range node.children {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		child := node.children[name]
		safeName := html.EscapeString(name)
		if child.isDir || len(child.children) > 0 {
			fmt.Fprintf(b, "<details%s>\n<summary class=\"dir\">%s/</summary>\n",
				openAttr(depth), safeName)
			tb.buildNode(b, child, depth+1)
			b.WriteString("</details>\n")
		} else {
			fmt.Fprintf(b, "<div class=\"file\">%s<span>%d</span></div>\n",
				safeName, child.size())
		}
	}
}

func openAttr(depth int) string {
	if depth == 0 {
		return " open"
	}
	return ""
}

func (n *treeNode) size() int64 {
	var total int64
	for _, child := range n.children {
		total += child.size()
	}
	return total
}
