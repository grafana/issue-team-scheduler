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
	"regexp"
	"testing"

	"github.com/go-kit/log"
	"github.com/google/go-github/github"
	"github.com/stretchr/testify/require"
)

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
