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
	"io"
	"os"
	"testing"
)

// TestVerify tests the Verify function
// You can use this to verify accessibility to members calendar locally. To do so:
// 1. Put the service account JSON secret in testdata/gcal-service-account.json
// 2. Add the members to the Config
// 3. Adjust your expectations and run the test
func TestVerify(t *testing.T) {
	loadServiceAccountToEnv(t)

	cfg := Config{
		Teams: map[string]TeamConfig{
			"team1": {
				Members: []MemberConfig{
					{
						Name:           "test",
						GoogleCalendar: "test@grafana.com",
						Output:         "test",
					},
				},
			},
		},
	}

	action := Action{
		Config: cfg,
	}

	im := action.Verify(context.TODO())
	if len(im) != 1 {
		t.Fatalf("Expected 1 inaccessible members, got %d", len(im))
	}
}

func loadServiceAccountToEnv(t *testing.T) {
	reader, err := os.Open("testdata/gcal-service-account.json")
	if err != nil {
		t.Fatalf("Unable to open service account JSON secret, due %v", err)
	}
	value, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Unable to read service account JSON secret, due %v", err)
	}
	os.Setenv("INPUT_GCAL-SERVICE-ACOUNT-KEY", string(value))
}
