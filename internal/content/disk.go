package content

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/yuin/goldmark"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
)

type Page struct {
	Id      string `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content,omitempty"`
}

func ReadAll() ([]Page, error) {
	var pages []Page
	files, err := filepath.Glob("data/content/*.md")
	if err != nil {
		return nil, fmt.Errorf("error reading /data/content/*.md: %w", err)
	}

	for _, file := range files {
		id := filepath.Base(file)
		id = id[0 : len(id)-3]
		page, err := Read(id)
		if err != nil {
			return nil, fmt.Errorf("error reading %s: %w", id, err)
		}
		pages = append(pages, page)
	}

	return pages, nil
}

func Read(id string) (Page, error) {
	data, err := os.ReadFile("data/content/" + id + ".md")
	if err != nil {
		return Page{}, fmt.Errorf("error reading /data/content/%s.md: %w", id, err)
	}

	markdown := goldmark.New(
		goldmark.WithExtensions(extension.GFM, meta.New()),
	)
	context := parser.NewContext()
	var buf bytes.Buffer
	if err := markdown.Convert(data, &buf, parser.WithContext(context)); err != nil {
		return Page{}, fmt.Errorf("error converting markdown: %w", err)
	}

	metaData := meta.Get(context)
	page := Page{
		Content: buf.String(),
		Id:      id,
		Title:   id,
	}
	if title, ok := metaData["title"]; ok {
		page.Title = title.(string)
	}

	return page, nil
}
