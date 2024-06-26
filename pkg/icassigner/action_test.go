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
	"fmt"
	"testing"
)

func TestFindTeam(t *testing.T) {
	teams := []TeamConfig{
		{
			RequireLabel: []string{"A", "B"},
			Members:      []MemberConfig{{Name: "1"}, {Name: "2"}},
		},
		{
			RequireLabel: []string{"C", "D"},
			Members:      []MemberConfig{{Name: "3"}, {Name: "4"}},
		},
	}
	testCases := []struct {
		name string

		teams []TeamConfig

		inputLabels []string

		expectedTeamMemberNames []string
	}{
		{
			teams:                   teams,
			inputLabels:             []string{"A", "B"},
			expectedTeamMemberNames: []string{"1", "2"},
		},
		{
			teams:                   teams,
			inputLabels:             []string{"B"},
			expectedTeamMemberNames: []string{"1", "2"},
		},
		{
			teams:                   teams,
			inputLabels:             []string{"E"},
			expectedTeamMemberNames: []string{},
		},
		{
			teams:                   teams,
			inputLabels:             []string{"C", "D"},
			expectedTeamMemberNames: []string{"3", "4"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			cfg := Config{Teams: map[string]TeamConfig{}}
			for i, team := range testCase.teams {
				cfg.Teams[fmt.Sprintf("%v", i)] = team
			}
			result, _ := findTeam(cfg, testCase.inputLabels)

			if len(testCase.expectedTeamMemberNames) == 0 && len(result) > 0 {
				t.Error("Expected to have no team members matching, but got", len(result))
			}

			for _, e := range testCase.expectedTeamMemberNames {
				found := false
				for _, a := range result {
					if e != a.Name {
						continue
					}

					found = true
					break
				}

				if !found {
					t.Errorf("Expected team members to contain %v, but only got %v", e, result)
				}
			}
		})
	}
}
