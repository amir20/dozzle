package releases

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

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
	Current       bool      `json:"current"`
}

func Fetch(currentVersion string) ([]Release, error) {
	response, err := http.Get("https://api.github.com/repos/amir20/dozzle/releases?per_page=12")
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
		releases = append(releases, Release{
			Name:          githubRelease.Name,
			MentionsCount: githubRelease.MentionsCount,
			Tag:           githubRelease.TagName,
			Body:          buffer.String(),
			CreatedAt:     githubRelease.CreatedAt,
			HtmlUrl:       githubRelease.HtmlUrl,
			Current:       githubRelease.TagName == currentVersion,
		})
		if githubRelease.TagName == currentVersion {
			break
		}
	}

	if len(releases) > 0 {
		releases[0].Latest = true
	}

	return releases, nil
}
