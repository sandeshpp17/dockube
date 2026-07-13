package markdown

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
)

type Rendered struct {
	Title, Owner string
	Tags         []string
	HTML         string
}

var linkRE = regexp.MustCompile(`\[\[([^\]|]+)(?:\|([^\]]+))?\]\]`)

func Render(source, product, version string) (Rendered, error) {
	meta, body := frontMatter(source)
	title := meta["title"]
	if title == "" {
		title = firstHeading(body)
		if title == "" {
			title = "Untitled"
		}
	}
	body = linkRE.ReplaceAllStringFunc(body, func(m string) string {
		p := linkRE.FindStringSubmatch(m)
		target := strings.TrimSpace(p[1])
		label := target
		if p[2] != "" {
			label = p[2]
		}
		target = strings.TrimSuffix(strings.TrimPrefix(target, "/"), ".md")
		return fmt.Sprintf("[%s](/docs/%s/%s/%s)", label, product, version, target)
	})
	// Mermaid remains a fenced code block; the renderer class is inert and scripts are never accepted from Markdown.
	md := goldmark.New(goldmark.WithExtensions(extension.GFM, highlighting.NewHighlighting()), goldmark.WithParserOptions(parser.WithAutoHeadingID()))
	var out bytes.Buffer
	if err := md.Convert([]byte(body), &out); err != nil {
		return Rendered{}, err
	}
	tags := split(meta["tags"])
	return Rendered{Title: title, Owner: meta["owner"], Tags: tags, HTML: out.String()}, nil
}
func frontMatter(s string) (map[string]string, string) {
	m := map[string]string{}
	if !strings.HasPrefix(s, "---\n") {
		return m, s
	}
	end := strings.Index(s[4:], "\n---")
	if end < 0 {
		return m, s
	}
	head := s[4 : 4+end]
	for _, l := range strings.Split(head, "\n") {
		p := strings.SplitN(l, ":", 2)
		if len(p) == 2 {
			m[strings.TrimSpace(p[0])] = strings.Trim(strings.TrimSpace(p[1]), "\"'")
		}
	}
	return m, s[4+end+5:]
}
func firstHeading(s string) string {
	for _, l := range strings.Split(s, "\n") {
		if strings.HasPrefix(l, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(l, "# "))
		}
	}
	return ""
}
func split(s string) []string {
	var r []string
	for _, v := range strings.Split(s, ",") {
		if v = strings.TrimSpace(v); v != "" {
			r = append(r, v)
		}
	}
	return r
}
