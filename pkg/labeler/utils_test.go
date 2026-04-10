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
	"testing"

	"github.com/google/go-github/github"
	"github.com/stretchr/testify/require"
)

func TestDecomposeRepoURL(t *testing.T) {
	testCases := []struct {
		url           string
		expectedOwner string
		expectedName  string
	}{
		{
			url:           "https://api.github.com/repos/grafana/grafana",
			expectedOwner: "grafana",
			expectedName:  "grafana",
		},
		{
			url:           "https://api.github.com/repos/my-org/my-repo",
			expectedOwner: "my-org",
			expectedName:  "my-repo",
		},
		{
			url:           "/repos/owner/repo",
			expectedOwner: "owner",
			expectedName:  "repo",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.url, func(t *testing.T) {
			owner, name := decomposeRepoURL(tc.url)
			require.Equal(t, tc.expectedOwner, owner)
			require.Equal(t, tc.expectedName, name)
		})
	}
}

func TestGetIssueLabels(t *testing.T) {
	t.Run("returns label names", func(t *testing.T) {
		issue := &github.Issue{
			Labels: []github.Label{
				{Name: github.String("bug")},
				{Name: github.String("enhancement")},
			},
		}
		require.Equal(t, []string{"bug", "enhancement"}, getIssueLabels(issue))
	})

	t.Run("returns empty slice for issue with no labels", func(t *testing.T) {
		issue := &github.Issue{Labels: []github.Label{}}
		require.Empty(t, getIssueLabels(issue))
	})
}
