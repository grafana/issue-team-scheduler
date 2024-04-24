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
	"context"
	"strings"

	"github.com/google/go-github/github"
)

func getLabelAssigner(gh *github.Client) labelAssigner {
	return func(repoOwner, repoName string, issueNumber int, labels []string) error {
		_, _, err := gh.Issues.ReplaceLabelsForIssue(context.Background(), repoOwner, repoName, issueNumber, labels)
		return err
	}
}

func getIssueLabels(issue *github.Issue) []string {
	issueLabels := make([]string, len(issue.Labels))

	for issueLabelIdx, issueLabel := range issue.Labels {
		if issueLabel.Name == nil {
			continue
		}
		issueLabels[issueLabelIdx] = *issueLabel.Name
	}

	return issueLabels
}

func decomposeRepoURL(repoURL string) (repoOwner, repoName string) {
	repoUrlSplit := strings.Split(repoURL, "/")
	repoName = repoUrlSplit[len(repoUrlSplit)-1]
	repoOwner = repoUrlSplit[len(repoUrlSplit)-2]
	return repoOwner, repoName
}
