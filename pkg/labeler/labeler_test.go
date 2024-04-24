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
