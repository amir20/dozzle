package releases

import (
	"bytes"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/yuin/goldmark/v2/ast"
	"github.com/yuin/goldmark/v2/parser"
	"github.com/yuin/goldmark/v2/renderer/html"
)

var (
	markdownParser   = parser.New()
	markdownRenderer = html.New()

	// releaseVersion matches a version Dozzle was released under, e.g. v11.1.0
	// or v11.1.0-beta.1. A dev build (main-<sha>, pr-123-<sha>, local) does not.
	releaseVersion = regexp.MustCompile(`^v?\d+\.\d+\.\d+([-+].*)?$`)
)

type githubRelease struct {
	Name          string    `json:"name"`
	MentionsCount int       `json:"mentions_count"`
	TagName       string    `json:"tag_name"`
	Body          string    `json:"body"`
	CreatedAt     time.Time `json:"created_at"`
	HtmlUrl       string    `json:"html_url"`
}

type Release struct {
	Name          string    `json:"name"`
	MentionsCount int       `json:"mentionsCount"`
	Tag           string    `json:"tag"`
	Body          string    `json:"body"`
	CreatedAt     time.Time `json:"createdAt"`
	HtmlUrl       string    `json:"htmlUrl"`
	Latest        bool      `json:"latest"`
	Features      int       `json:"features"`
	BugFixes      int       `json:"bugFixes"`
	Breaking      int       `json:"breaking"`
}

// OnReleaseLine reports whether version is one Dozzle was released under, and
// so whether comparing it against a release tag means anything.
func OnReleaseLine(version string) bool {
	return releaseVersion.MatchString(version)
}

func Fetch(currentVersion string) ([]Release, error) {
	response, err := http.Get("https://api.github.com/repos/amir20/dozzle/releases?per_page=9")
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	var githubReleases []githubRelease
	if err := json.NewDecoder(response.Body).Decode(&githubReleases); err != nil {
		return []Release{}, err
	}

	var releases []Release
	for _, githubRelease := range githubReleases {
		if githubRelease.TagName == currentVersion {
			break
		}

		html, features, bugFixes, breaking := parseBody([]byte(githubRelease.Body))
		release := Release{
			Name:          githubRelease.Name,
			MentionsCount: githubRelease.MentionsCount,
			Tag:           githubRelease.TagName,
			Body:          html,
			CreatedAt:     githubRelease.CreatedAt,
			HtmlUrl:       githubRelease.HtmlUrl,
			Features:      features,
			BugFixes:      bugFixes,
			Breaking:      breaking,
		}

		releases = append(releases, release)
	}

	// Latest means "newer than what you run", which only has an answer on the
	// release line. A dev build (main-<sha>, a PR image, a local build) sits
	// off it and matches no tag, so every release would otherwise read as an
	// update waiting to be installed. The notes still list, they just stop
	// claiming that.
	if len(releases) > 0 && OnReleaseLine(currentVersion) {
		releases[0].Latest = true
	}

	return releases, nil
}

// parseBody renders a release body to HTML and counts the items listed under
// its Features, Bug Fixes and Breaking Changes headings: every list item
// (nested ones included) in the block that follows an h3 whose text contains
// the section name. A later matching heading overwrites an earlier one.
func parseBody(body []byte) (html string, features, bugFixes, breaking int) {
	doc := markdownParser.Parse(body)
	var buffer bytes.Buffer
	markdownRenderer.Render(&buffer, body, doc)

	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		heading, ok := n.(*ast.Heading)
		if !entering || !ok || heading.Level != 3 {
			return ast.WalkContinue, nil
		}
		title := plainText(heading, body)
		items := listItems(nextRendered(heading))
		if strings.Contains(title, "Features") {
			features = items
		}
		if strings.Contains(title, "Bug Fixes") {
			bugFixes = items
		}
		if strings.Contains(title, "Breaking Changes") {
			breaking = items
		}
		return ast.WalkSkipChildren, nil
	})

	return buffer.String(), features, bugFixes, breaking
}

// plainText is the text a reader sees in n. Image alt text renders as an
// attribute, not text, so it is left out.
func plainText(n ast.Node, source []byte) string {
	var sb strings.Builder
	ast.Walk(n, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n := n.(type) {
		case *ast.Image:
			return ast.WalkSkipChildren, nil
		case *ast.Text:
			sb.WriteString(n.Value.Value(source))
		case *ast.CodeSpan:
			sb.WriteString(n.Value.Value(source))
		case *ast.AutoLink:
			sb.WriteString(n.Label.Value(source))
		}
		return ast.WalkContinue, nil
	})
	return sb.String()
}

// nextRendered is the next sibling of n that renders as an element. Raw HTML
// blocks render as a comment and link reference definitions as nothing.
func nextRendered(n ast.Node) ast.Node {
	for next := n.NextSibling(); next != nil; next = next.NextSibling() {
		switch next.Kind() {
		case ast.KindHTMLBlock, ast.KindLinkReferenceDefinition:
			continue
		}
		return next
	}
	return nil
}

// listItems counts the list items below n, not counting n itself.
func listItems(n ast.Node) int {
	if n == nil {
		return 0
	}
	count := 0
	ast.Walk(n, func(child ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering && child != n && child.Kind() == ast.KindListItem {
			count++
		}
		return ast.WalkContinue, nil
	})
	return count
}
