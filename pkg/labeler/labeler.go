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

package labeler

import (
	"errors"
	"slices"
	"strings"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/go-github/github"
	githubaction "github.com/grafana/escalation-scheduler/pkg/github-action"
)

type labelAssigner func(repoOwner, repoName string, issueNumber int, labels []string) error

type Labeler struct {
	cfg           Config
	labelAssigner labelAssigner
	logger        log.Logger
}

func NewLabeler(cfg Config, gh *github.Client, logger log.Logger) *Labeler {
	return &Labeler{
		cfg:           cfg,
		labelAssigner: getLabelAssigner(gh),
		logger:        logger,
	}
}

func (l *Labeler) Run(issue *github.Issue) error {
	if !l.hasRequiredLabels(issue) {
		level.Info(l.logger).Log("msg", "issue has none of the required labels", "requireLabel", strings.Join(l.cfg.RequireLabel, ", "))
		return nil
	}

	level.Info(l.logger).Log("msg", "issue has at least one of the required labels", "requireLabel", strings.Join(l.cfg.RequireLabel, ", "))

	if l.hasAssignableLabel(issue) {
		return nil
	}

	level.Info(l.logger).Log("msg", "issue does not have one of the assignable labels", "assignable_labels", strings.Join(l.getAssignableLabels(), ", "))

	label, err := l.findLabel(issue.GetTitle(), issue.GetBody())
	if err != nil {
		return err
	}

	err = l.assignLabel(issue, label)
	if err != nil {
		return err
	}

	err = githubaction.SetOutput("assignedLabel", label)
	if err != nil {
		return err
	}

	level.Info(l.logger).Log("msg", "completed successfully")

	return nil
}

func (l *Labeler) findLabel(title, body string) (label string, err error) {
	// Don't log title / body because they might contain sensitive data.

	scoreByLabel := make(map[string]int)
	for label, properties := range l.cfg.Labels {
		level.Info(l.logger).Log("msg", "evaluating regular expressions for label", "label", label)

		for _, matcher := range properties.Matchers {
			if matcher.regex.MatchString(title) || matcher.regex.MatchString(body) {
				level.Info(l.logger).Log("msg", "regex matches", "regex", matcher.RegexStr, "weight", matcher.Weight)
				scoreByLabel[label] += matcher.Weight
			} else {
				level.Info(l.logger).Log("msg", "regex does not match", "regex", matcher.RegexStr)
			}
		}
	}

	var bestLabel string
	for label, score := range scoreByLabel {
		level.Info(l.logger).Log("msg", "label has score assigned", "label", label, "score", score)
		if bestLabel == "" {
			bestLabel = label
			continue
		}

		if score > scoreByLabel[bestLabel] {
			bestLabel = label
		}
	}

	if scoreByLabel[bestLabel] == 0 {
		return "", errors.New("no label found")
	}

	level.Info(l.logger).Log("msg", "label has been chosen", "label", bestLabel, "score", scoreByLabel[bestLabel])

	return bestLabel, nil
}

func (l *Labeler) assignLabel(issue *github.Issue, label string) error {
	level.Info(l.logger).Log("msg", "assigning label to issue", "label", label)

	issueLabels := getIssueLabels(issue)
	level.Info(l.logger).Log("msg", "issue currently has labels", "labels", strings.Join(issueLabels, ", "))

	for _, currentLabel := range issueLabels {
		if currentLabel == label {
			level.Info(l.logger).Log("msg", "issue already has the label", "label", label)
			// Label already assigned
			return nil
		}
	}

	issueLabels = append(issueLabels, label)

	level.Info(l.logger).Log("msg", "issue is going to have labels", "labels", strings.Join(issueLabels, ", "))

	repoOwner, repoName := decomposeRepoURL(*issue.RepositoryURL)
	return l.labelAssigner(repoOwner, repoName, issue.GetNumber(), issueLabels)
}

func (l *Labeler) getAssignableLabels() []string {
	assignableLabels := make([]string, 0, len(l.cfg.Labels))
	for assignableLabel := range l.cfg.Labels {
		assignableLabels = append(assignableLabels, assignableLabel)
	}
	return assignableLabels
}

func (l *Labeler) hasAssignableLabel(issue *github.Issue) bool {
	issueLabels := getIssueLabels(issue)

	for _, assignableLabel := range l.getAssignableLabels() {
		if slices.Contains[[]string, string](issueLabels, assignableLabel) {
			level.Info(l.logger).Log("msg", "issue already has at least one of the assignable labels, aborting run", "label", assignableLabel)
			return true
		}
	}

	return false
}

func (l *Labeler) hasRequiredLabels(issue *github.Issue) bool {
	issueLabels := getIssueLabels(issue)

	match := false
	for _, requiredLabel := range l.cfg.RequireLabel {
		if slices.Contains[[]string, string](issueLabels, requiredLabel) {
			match = true
		}

	}

	return match
}
