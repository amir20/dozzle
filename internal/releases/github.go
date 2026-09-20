package releases

import (
	"bytes"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/yuin/goldmark/v2/parser"
	"github.com/yuin/goldmark/v2/renderer/html"
)

var (
	markdownParser   = parser.New()
	markdownRenderer = html.New()

	// releaseVersion matches a version Dozzle was released under, e.g. v11.1.0
	// or v11.1.0-beta.1. A dev build (master-<sha>, pr-123-<sha>, local) does not.
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
		var buffer bytes.Buffer
		body := []byte(githubRelease.Body)
		markdownRenderer.Render(&buffer, body, markdownParser.Parse(body))
		html := buffer.String()

		if githubRelease.TagName == currentVersion {
			break
		}

		release := Release{
			Name:          githubRelease.Name,
			MentionsCount: githubRelease.MentionsCount,
			Tag:           githubRelease.TagName,
			Body:          html,
			CreatedAt:     githubRelease.CreatedAt,
			HtmlUrl:       githubRelease.HtmlUrl,
		}

		doc, _ := goquery.NewDocumentFromReader(&buffer)
		doc.Find("h3").Each(func(i int, s *goquery.Selection) {
			if strings.Contains(s.Text(), "Features") {
				release.Features = s.Next().Find("li").Length()
			}

			if strings.Contains(s.Text(), "Bug Fixes") {
				release.BugFixes = s.Next().Find("li").Length()
			}

			if strings.Contains(s.Text(), "Breaking Changes") {
				release.Breaking = s.Next().Find("li").Length()
			}
		})

		releases = append(releases, release)
	}

	// Latest means "newer than what you run", which only has an answer on the
	// release line. A dev build (master-<sha>, a PR image, a local build) sits
	// off it and matches no tag, so every release would otherwise read as an
	// update waiting to be installed. The notes still list, they just stop
	// claiming that.
	if len(releases) > 0 && OnReleaseLine(currentVersion) {
		releases[0].Latest = true
	}

	return releases, nil
}
