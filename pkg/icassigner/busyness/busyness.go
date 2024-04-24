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

package busyness

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/google/go-github/github"
	githubaction "github.com/grafana/escalation-scheduler/pkg/github-action"
)

// Report represents the busyness of a team in ascending order of busyness
type Report []Level

// Level represents one level of busyness and all team members which match the given busyness
type Level struct {
	Busyness int
	Users    []string
}

// busynessClient is used to determine the busyness of a given team member as of today since the specified time
type busynessClient interface {
	// getBusyness returns the busyness of a given user since a given time
	getBusyness(ctx context.Context, since time.Time, user string) int
}

// CalculateBusynessForTeam calculates busyness of all members and returns a BusynessReport for them
func CalculateBusynessForTeam(ctx context.Context, now time.Time, githubClient *github.Client, ignorableLabels []string, members []string) (Report, error) {
	bA, err := newGithubBusynessClient(githubClient, ignorableLabels)
	if err != nil {
		return Report{}, fmt.Errorf("unable to create github busyness client, due %w", err)
	}
	return calculateBusynessForTeam(ctx, now, bA, members), nil
}

func calculateBusynessForTeam(ctx context.Context, now time.Time, bA busynessClient, members []string) Report {
	// calculate lookback time. If we are on a time shortly after the weekend we'll extend to lookback by 2 days to account for weekends
	// TODO: Should this be configurable?
	since := now.Add(-5 * 24 * time.Hour)
	if since.Weekday() == time.Saturday || since.Weekday() == time.Sunday {
		since = since.Add(-2 * 24 * time.Hour)
	}

	addMember := func(b map[int][]string, m string, busyness int) {
		v, ok := b[busyness]
		if !ok {
			v = []string{m}
		} else {
			v = append(v, m)
		}

		b[busyness] = v
	}

	// get busyness by team member
	busyness := map[int][]string{}
	for _, member := range members {
		b := bA.getBusyness(ctx, since, member)
		addMember(busyness, member, b)
	}

	// transform map into array
	report := make([]Level, 0, len(busyness))
	for b, members := range busyness {
		report = append(report, Level{Busyness: b, Users: members})
	}

	// sort in ascending order
	slices.SortFunc[[]Level](report, func(a, b Level) int {
		return a.Busyness - b.Busyness
	})

	return report
}

// githubBusynessClient is used to calculate busyness of users by the amount of github issues they are assigned to
type githubBusynessClient struct {
	labelsToIgnore map[string]struct{}

	listByAssigneeFunc func(ctx context.Context, since time.Time, assignee string, amount int) ([]*github.Issue, error)
}

// newGithubBusynessClient creates a new githubBusynessClient based on config.
func newGithubBusynessClient(githubClient *github.Client, ignorableLabels []string) (*githubBusynessClient, error) {
	labelsToIgnore := map[string]struct{}{}
	for _, i := range ignorableLabels {
		labelsToIgnore[i] = struct{}{}
	}

	owner, repo, _, err := githubaction.Repository()
	if err != nil {
		return nil, fmt.Errorf("unable to get github repository information due %w", err)
	}
	listByAssigneeFunc := func(ctx context.Context, since time.Time, assignee string, amount int) ([]*github.Issue, error) {
		issues, _, err := githubClient.Issues.ListByRepo(ctx, owner, repo, &github.IssueListByRepoOptions{
			Since:    since,    // check only issues which were updated since
			Assignee: assignee, // filter by assignee
			ListOptions: github.ListOptions{
				PerPage: amount, // we only want as many as specified in amount
			},
			Sort: "updated", // sort descending by last updated
		})

		return issues, err
	}

	return &githubBusynessClient{
		labelsToIgnore:     labelsToIgnore,
		listByAssigneeFunc: listByAssigneeFunc,
	}, nil
}

// getBusyness returns the busyness of a given team member since the specified time.
// Busyness is the count of all open issues assigned to the team member plus all issues closed after the specified time in "since".
//
// If there are labels to be ignored, all issues with that label are ignored from the busyness calculation
func (b *githubBusynessClient) getBusyness(ctx context.Context, since time.Time, member string) int {
	// check if one of the labels is contained by the labels to ignore
	containsLabelsToIgnore := func(labels []github.Label) bool {
		for _, l := range labels {
			_, tobeIgnored := b.labelsToIgnore[*l.Name]
			if tobeIgnored {
				return true
			}
		}
		return false
	}

	issues, err := b.listByAssigneeFunc(ctx, since, member, 20)
	if err != nil {
		return 0
	}

	// count relevant issues
	busyness := 0
	for _, i := range issues {
		// ignore everything without any state
		if i.State == nil {
			continue
		}

		switch *i.State {
		case "open":
			// check for labels to ignore, e.g. `stale` and ignore issue in this case
			if containsLabelsToIgnore(i.Labels) {
				continue
			}

			// increase busyness count otherwise
			busyness++
		case "closed":
			// if the issue got closed since our time to check
			if since.Before(*i.ClosedAt) {
				busyness++
			}
		}
	}

	return busyness
}
