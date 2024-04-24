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
