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
	"context"
	"log"
	"strings"

	githubaction "github.com/grafana/escalation-scheduler/pkg/github-action"
)

// Verify the calendar access of everyone defined in the configuration.
//
// All members of whom calendars can't be accessed are set as "inaccessibleMembers" output
func (a *Action) Verify(ctx context.Context) []string {
	inaccessibleMembers := []string{}

	for _, team := range a.Config.Teams {
		for _, m := range team.Members {

			_, err := checkAvailability(m)
			if err != nil {
				log.Printf("Unable to check availability of %q, due %v\n", m.Name, err)
				inaccessibleMembers = append(inaccessibleMembers, m.Name)
			} else {
				log.Printf("Able to check availability of %q\n", m.Name)
			}

		}
	}

	githubaction.SetOutput("inaccessibleMembers", strings.Join(inaccessibleMembers, ", "))

	return inaccessibleMembers
}
