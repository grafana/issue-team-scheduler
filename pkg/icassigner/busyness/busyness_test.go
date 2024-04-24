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
	"errors"
	"testing"
	"time"

	"github.com/google/go-github/github"
)

func TestCalculateIssueBusynessPerTeamMember(t *testing.T) {
	ctx := context.TODO()

	now := time.Now()

	members := []string{
		"IC 1",
		"IC 2",
		"IC 3",
	}

	testcases := []struct {
		name string

		time              time.Time
		busynessPerMember map[string]int

		expectedReport Report
	}{
		{
			name: "TestDefaultCaseWorks",
			time: now,
			busynessPerMember: map[string]int{
				members[0]: 1,
				members[1]: 3,
				members[2]: 3,
			},
			expectedReport: Report{
				Level{Busyness: 1, Users: []string{members[0]}},
				Level{Busyness: 3, Users: []string{members[1], members[2]}},
			},
		},
		{
			name: "TestDefaultCaseWorks2",
			time: now,
			busynessPerMember: map[string]int{
				members[0]: 3,
				members[1]: 0,
				members[2]: 3,
			},
			expectedReport: Report{
				Level{Busyness: 0, Users: []string{members[1]}},
				Level{Busyness: 3, Users: []string{members[0], members[2]}},
			},
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			busynessClient := &mockBusynessClient{
				inputs:             make(map[string]time.Time),
				resultByMemberName: testcase.busynessPerMember,
			}

			report := calculateBusynessForTeam(ctx, testcase.time, busynessClient, members)

			if len(testcase.expectedReport) != len(report) {
				t.Fatalf("Expected same levels of busyness of %v, but got %v", testcase.expectedReport, report)
			}

			for i, busynessLevel := range report {
				expectedLevel := testcase.expectedReport[i]

				if expectedLevel.Busyness != busynessLevel.Busyness {
					t.Errorf("Expected same level of busyness %v at index %v, but got %v", expectedLevel.Busyness, i, busynessLevel.Busyness)
				}

				if len(expectedLevel.Users) != len(busynessLevel.Users) {
					t.Errorf("Expected %v members at busyness level %v, but got %v", len(expectedLevel.Users), expectedLevel.Busyness, len(busynessLevel.Users))
					continue // no value in checking individual members at this point
				}

				for j, member := range busynessLevel.Users {
					expectedMember := expectedLevel.Users[j]

					if expectedMember != member {
						t.Errorf("Expected member at level %v and index %v to be %q, but got %q", busynessLevel.Busyness, j, expectedMember, member)
					}
				}
			}
		})
	}
}

// TestGithubBusynessClient_getBusyness tests that we get the right busyness based on github issues for team member.
func TestGithubBusynessClient_getBusyness(t *testing.T) {
	ctx := context.TODO()

	since := time.Now() // we don't need a real "since" as it's directly passed to the issue client
	member := "Test"    // we don't need a real "member" as it's simply forwarded to the issue client

	str := func(s string) *string { return &s }

	testcases := []struct {
		Name string // Name of the test

		IgnoredLabels           []string                        // Labels which tag issues which be ignored
		MockedIssueClientResult func() ([]*github.Issue, error) // mocked result of the github client

		ExpectedBusyness int // expected result based on mocked result and labels to ignore
	}{
		{
			Name:                    "TestNoBusynessIsReportedInCaseOfError",
			MockedIssueClientResult: func() ([]*github.Issue, error) { return nil, errors.New("error") },
			ExpectedBusyness:        0,
		},
		{
			Name: "TestNormalTestcaseWorks",
			MockedIssueClientResult: func() ([]*github.Issue, error) {
				return []*github.Issue{
					{
						State: str("open"),
					},
				}, nil
			},
			ExpectedBusyness: 1,
		},
		{
			Name: "TestMultipleIssuesAreCounted",
			MockedIssueClientResult: func() ([]*github.Issue, error) {
				return []*github.Issue{
					{
						State: str("open"),
					},
					{
						State: str("open"),
					},
				}, nil
			},
			ExpectedBusyness: 2,
		},
		{
			Name: "TestIssuesAreCountedIfClosedAfterSince",
			MockedIssueClientResult: func() ([]*github.Issue, error) {
				closedTime := since.Add(1 * time.Minute)
				return []*github.Issue{
					{
						State:    str("closed"),
						ClosedAt: &closedTime,
					},
				}, nil
			},
			ExpectedBusyness: 1,
		},
		{
			Name: "TestClosedIssuesAreNotCountedIfClosedBeforeSince",
			MockedIssueClientResult: func() ([]*github.Issue, error) {
				closedTime := since.Add(-1 * time.Minute)
				return []*github.Issue{
					{
						State:    str("closed"),
						ClosedAt: &closedTime,
					},
				}, nil
			},
			ExpectedBusyness: 0,
		},
		{
			Name:          "TestIssuesAreNotCountedIfLabelIsToBeIgnored",
			IgnoredLabels: []string{"stale"},
			MockedIssueClientResult: func() ([]*github.Issue, error) {
				return []*github.Issue{
					{
						State:  str("open"),
						Labels: []github.Label{{Name: str("stale")}},
					},
				}, nil
			},
			ExpectedBusyness: 0,
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.Name, func(t *testing.T) {
			mock := &mockIssueClient{}
			ba := &githubBusynessClient{
				listByAssigneeFunc: mock.ListByAssignee,
				labelsToIgnore:     make(map[string]struct{}),
			}

			for _, v := range testcase.IgnoredLabels {
				ba.labelsToIgnore[v] = struct{}{}
			}

			mockResult, mockErr := testcase.MockedIssueClientResult()
			mock.result.issues = mockResult
			mock.result.err = mockErr

			busyness := ba.getBusyness(ctx, since, member)

			if mock.input.assignee != member {
				t.Error("Expected assignee passed to issue client to be", member, ", but got", mock.input.assignee)
			}

			if mock.input.since != since {
				t.Error("Expected since passed to issue client to be", since, ", but got", mock.input.since)
			}

			if busyness != testcase.ExpectedBusyness {
				t.Error("Expected busyness to be", testcase.ExpectedBusyness, ", but got", busyness)
			}
		})
	}
}

type mockIssueClient struct {
	input struct {
		assignee string
		since    time.Time
		amount   int
	}

	result struct {
		issues []*github.Issue
		err    error
	}
}

func (m *mockIssueClient) ListByAssignee(ctx context.Context, since time.Time, assignee string, amount int) ([]*github.Issue, error) {
	m.input.since = since
	m.input.assignee = assignee
	m.input.amount = amount

	return m.result.issues, m.result.err
}

type mockBusynessClient struct {
	inputs map[string]time.Time

	resultByMemberName map[string]int
}

func (m *mockBusynessClient) getBusyness(ctx context.Context, since time.Time, member string) int {
	m.inputs[member] = since

	return m.resultByMemberName[member]
}
