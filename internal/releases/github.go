package releases

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/yuin/goldmark"
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
		goldmark.Convert([]byte(githubRelease.Body), &buffer)
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

	if len(releases) > 0 {
		releases[0].Latest = true
	}

	return releases, nil
}
