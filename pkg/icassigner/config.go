package icassigner

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/google/go-github/github"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Teams         map[string]TeamConfig `yaml:"teams,omitempty"`
	IgnoredLabels []string              `yaml:"ignoreLabels,omitempty"`
}

type TeamConfig struct {
	RequireLabel []string       `yaml:"requireLabel,omitempty"`
	Members      []MemberConfig `yaml:"members,omitempty"`
}

type MemberConfig struct {
	Name           string `yaml:"name,omitempty"`
	IcalURL        string `yaml:"ical-url,omitempty"`
	GoogleCalendar string `yaml:"googleCalendar,omitempty"`
	Output         string `yaml:"output,omitempty"`
}

func ParseConfig(r io.Reader) (Config, error) {
	var cfg Config

	err := yaml.NewDecoder(r).Decode(&cfg)
	if err != nil {
		return cfg, fmt.Errorf("unable to parse config, due: %w", err)
	}

	return cfg, nil
}

func FetchConfig(ctx context.Context, client *github.Client, owner, repo, ref, path string) (io.Reader, error) {
	rawContent, _, _, err := client.Repositories.GetContents(ctx, owner, repo, path, &github.RepositoryContentGetOptions{
		Ref: ref,
	})

	if err != nil {
		return nil, fmt.Errorf("unable to retrieve config, due %w", err)
	}

	content, err := rawContent.GetContent()
	if err != nil {
		return nil, fmt.Errorf("unable to load config, due %w", err)
	}

	return bytes.NewBuffer([]byte(content)), nil
}
