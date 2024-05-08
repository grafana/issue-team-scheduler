// SPDX-License-Identifier: Apache-2.0
//
// Copyright 2024 Grafana Labs
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package icassigner

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/google/go-github/github"
	"github.com/grafana/escalation-scheduler/pkg/icassigner/calendar"
	"gopkg.in/yaml.v2"
)

type Config struct {
	UnavailabilityLimit time.Duration         `yaml:"unavailabilityLimit,omitempty"`
	Teams               map[string]TeamConfig `yaml:"teams,omitempty"`
	IgnoredLabels       []string              `yaml:"ignoreLabels,omitempty"`
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

	if cfg.UnavailabilityLimit == 0 {
		// If unset, set default value
		cfg.UnavailabilityLimit = calendar.DefaultUnavailabilityLimit
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
