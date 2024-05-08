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
	"bytes"
	"testing"
	"time"
)

func TestMimirConfigCanBeParsed(t *testing.T) {
	// We usually don't want to test libraries we are using (yaml decoding in this case).
	// Nevertheless it was decided to add a test to ensure that loading of configs which were written for the js based version, still work.

	rawConfig := `teams:
  mimir:
    requireLabel:
    - cloud-prometheus
    - enterprise-metrics
    members:
    - name: tester1
      ical-url: https://tester1/basic.ics
      output: slack1
    - name: tester2
      ical-url: https://tester2/basic.ics
      output: slack2
ignoreLabels:
- stale
unavailabilityLimit: 6h` // redacted excerpt from a real world config

	r := bytes.NewBuffer([]byte(rawConfig))

	cfg, err := ParseConfig(r)

	if err != nil {
		t.Fatal("Error from config parsing should be nil, but got", err)
	}

	if cfg.IgnoredLabels == nil || len(cfg.IgnoredLabels) != 1 || cfg.IgnoredLabels[0] != "stale" {
		t.Error("Expected to get 1 ignore label `stale`, but got", cfg.IgnoredLabels)
	}

	team, ok := cfg.Teams["mimir"]
	if !ok {
		t.Fatal("Expected to find team \"mimir\", but got none")
	}

	if cfg.UnavailabilityLimit != 6*time.Hour {
		t.Error("Expected unavailability limit to be 6h, but got", cfg.UnavailabilityLimit)
	}

	expectedRequiredLabels := []string{"cloud-prometheus", "enterprise-metrics"}
	for i, e := range expectedRequiredLabels {
		if i >= len(team.RequireLabel) {
			t.Error("Expected require label at index", i, "but only got", len(team.RequireLabel))
			continue
		}

		if e != team.RequireLabel[i] {
			t.Error("Expected require label at index", i, "to be", e, ", but got:", team.RequireLabel[i])
		}
	}

	expectedMembers := []MemberConfig{
		{
			Name:    "tester1",
			IcalURL: "https://tester1/basic.ics",
			Output:  "slack1",
		},
		{
			Name:    "tester2",
			IcalURL: "https://tester2/basic.ics",
			Output:  "slack2",
		},
	}
	for i, e := range expectedMembers {
		if i >= len(team.Members) {
			t.Error("Expected team member at index", i, "but only got", len(team.Members))
			continue
		}

		actualMember := team.Members[i]

		if e.Name != actualMember.Name {
			t.Error("Expected team member name to be", e.Name, "but got", actualMember.Name)
		}

		if e.IcalURL != actualMember.IcalURL {
			t.Error("Expected team member ical url to be", e.IcalURL, "but got", actualMember.IcalURL)
		}

		if e.Output != actualMember.Output {
			t.Error("Expected team member outbut to be", e.Output, "but got", actualMember.Output)
		}
	}

}
