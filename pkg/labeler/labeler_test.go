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
	"os"
	"regexp"
	"testing"

	"github.com/go-kit/log"
	"github.com/google/go-github/github"
	"github.com/stretchr/testify/require"
)

// setGithubOutput creates a temp file for GITHUB_OUTPUT and registers cleanup.
func setGithubOutput(t *testing.T) {
	t.Helper()
	f, err := os.CreateTemp("", "github-output-*")
	require.NoError(t, err)
	t.Cleanup(func() { os.Remove(f.Name()) })
	f.Close()
	t.Setenv("GITHUB_OUTPUT", f.Name())
}

func TestAssigningLabel(t *testing.T) {
	testIssueNumber := 333
	testRepoOwner := "testOwner"
	testRepoName := "testRepo"
	testRepositoreURL := "/repos/" + testRepoOwner + "/" + testRepoName

	type testCase struct {
		name                       string
		cfg                        Config
		issue                      *github.Issue
		expectedLabelAssignerCalls []labelAssignerCall
	}

	testCases := []testCase{
		{
			name: "assign the mimir-query label",
			cfg: Config{
				RequireLabel: []string{"label1", "label3"},
				Labels: map[string]Label{
					"mimir-ingest": {
						Matchers: []Matcher{
							{
								regex:  regexp.MustCompile(`.*something something ingest.*`),
								Weight: 1,
							},
						},
					},
					"mimir-query": {
						Matchers: []Matcher{
							{
								regex:  regexp.MustCompile(`.*something something query.*`),
								Weight: 1,
							},
						},
					},
				},
			},
			issue: &github.Issue{
				Number:        &testIssueNumber,
				Title:         github.String("some title"),
				Body:          github.String("some body abc something something query more text."),
				RepositoryURL: &testRepositoreURL,
				Labels: []github.Label{
					{Name: github.String("label1")},
					{Name: github.String("label2")},
				},
			},
			expectedLabelAssignerCalls: []labelAssignerCall{{
				repoOwner:   testRepoOwner,
				repoName:    testRepoName,
				issueNumber: testIssueNumber,
				labels:      []string{"label1", "label2", "mimir-query"},
			}},
		}, {
			name: "don't assign label due to lack of required labels",
			cfg: Config{
				RequireLabel: []string{"label1", "label2"},
				Labels: map[string]Label{
					"mimir-query": {
						Matchers: []Matcher{
							{
								regex:  regexp.MustCompile(`.*`),
								Weight: 1,
							},
						},
					},
				},
			},
			issue: &github.Issue{
				Number:        &testIssueNumber,
				Title:         github.String("some title"),
				Body:          github.String("some body abc something something query more text."),
				RepositoryURL: &testRepositoreURL,
				Labels: []github.Label{
					{Name: github.String("label3")},
					{Name: github.String("label4")},
				},
			},
			expectedLabelAssignerCalls: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			setGithubOutput(t)
			mockLabelAssigner, calls := getMockLabelAssigner()
			l := &Labeler{
				cfg:           tc.cfg,
				labelAssigner: mockLabelAssigner,
				logger:        log.NewNopLogger(),
			}
			require.NoError(t, l.Run(tc.issue))

			require.Equal(t, tc.expectedLabelAssignerCalls, *calls)
		})
	}
}

func TestAssigningLabel_AlreadyHasAssignableLabel(t *testing.T) {
	testIssueNumber := 333
	testRepoOwner := "testOwner"
	testRepoName := "testRepo"
	testRepositoryURL := "/repos/" + testRepoOwner + "/" + testRepoName

	cfg := Config{
		RequireLabel: []string{"required"},
		Labels: map[string]Label{
			"target-label": {
				Matchers: []Matcher{
					{regex: regexp.MustCompile(`.*`), Weight: 1},
				},
			},
		},
	}
	issue := &github.Issue{
		Number:        &testIssueNumber,
		Title:         github.String("some title"),
		Body:          github.String("some body"),
		RepositoryURL: &testRepositoryURL,
		Labels: []github.Label{
			{Name: github.String("required")},
			{Name: github.String("target-label")}, // already has the assignable label
		},
	}

	mockLabelAssigner, calls := getMockLabelAssigner()
	l := &Labeler{cfg: cfg, labelAssigner: mockLabelAssigner, logger: log.NewNopLogger()}
	require.NoError(t, l.Run(issue))
	require.Nil(t, *calls, "expected no label assigner calls when issue already has an assignable label")
}

func TestAssigningLabel_HigherWeightedScoreWins(t *testing.T) {
	setGithubOutput(t)

	testIssueNumber := 333
	testRepoOwner := "testOwner"
	testRepoName := "testRepo"
	testRepositoryURL := "/repos/" + testRepoOwner + "/" + testRepoName

	cfg := Config{
		RequireLabel: []string{"required"},
		Labels: map[string]Label{
			"low-priority": {
				Matchers: []Matcher{
					{regex: regexp.MustCompile(`.*query.*`), Weight: 1},
				},
			},
			"high-priority": {
				Matchers: []Matcher{
					{regex: regexp.MustCompile(`.*query.*`), Weight: 2},
				},
			},
		},
	}
	issue := &github.Issue{
		Number:        &testIssueNumber,
		Title:         github.String("a query title"),
		Body:          github.String("some body"),
		RepositoryURL: &testRepositoryURL,
		Labels:        []github.Label{{Name: github.String("required")}},
	}

	mockLabelAssigner, calls := getMockLabelAssigner()
	l := &Labeler{cfg: cfg, labelAssigner: mockLabelAssigner, logger: log.NewNopLogger()}
	require.NoError(t, l.Run(issue))

	require.Len(t, *calls, 1)
	require.Contains(t, (*calls)[0].labels, "high-priority")
	require.NotContains(t, (*calls)[0].labels, "low-priority")
}

func TestFindLabel_NoMatch(t *testing.T) {
	l := &Labeler{
		cfg: Config{
			Labels: map[string]Label{
				"some-label": {
					Matchers: []Matcher{
						{regex: regexp.MustCompile(`very specific text that will not match`), Weight: 1},
					},
				},
			},
		},
		logger: log.NewNopLogger(),
	}

	_, err := l.findLabel("unrelated title", "unrelated body")
	require.Error(t, err, "expected error when no regex matches")
}

type labelAssignerCall struct {
	repoOwner   string
	repoName    string
	issueNumber int
	labels      []string
}

func getMockLabelAssigner() (labelAssigner, *[]labelAssignerCall) {
	var calls []labelAssignerCall
	return func(repoOwner, repoName string, issueNumber int, labels []string) error {
		calls = append(calls, labelAssignerCall{
			repoOwner:   repoOwner,
			repoName:    repoName,
			issueNumber: issueNumber,
			labels:      labels,
		})
		return nil
	}, &calls
}
