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

	"github.com/stretchr/testify/require"
)

func TestParseConfig(t *testing.T) {
	raw := []byte(`
requireLabel:
  - escalation
labels:
  squad-a:
    matchers:
      - regex: '.*query.*'
        weight: 2
      - regex: '.*read.*'
  squad-b:
    matchers:
      - regex: '.*ingest.*'
`)
	cfg, err := ParseConfig(raw)
	require.NoError(t, err)
	require.Equal(t, []string{"escalation"}, cfg.RequireLabel)
	require.Len(t, cfg.Labels, 2)

	squadA, ok := cfg.Labels["squad-a"]
	require.True(t, ok, "expected squad-a label to be present")
	require.Len(t, squadA.Matchers, 2)
	require.Equal(t, `.*query.*`, squadA.Matchers[0].RegexStr)
	require.Equal(t, 2, squadA.Matchers[0].Weight)
	require.Equal(t, `.*read.*`, squadA.Matchers[1].RegexStr)
}

func TestConfig_Validate_SetsDefaultWeight(t *testing.T) {
	cfg := Config{
		Labels: map[string]Label{
			"label": {
				Matchers: []Matcher{
					{RegexStr: `.*`, Weight: 0}, // zero weight should default to 1
				},
			},
		},
	}

	require.NoError(t, cfg.Validate())
	require.Equal(t, 1, cfg.Labels["label"].Matchers[0].Weight)
}

func TestConfig_Validate_CompilesRegex(t *testing.T) {
	cfg := Config{
		Labels: map[string]Label{
			"label": {
				Matchers: []Matcher{
					{RegexStr: `.*query.*`, Weight: 1},
				},
			},
		},
	}

	require.NoError(t, cfg.Validate())

	// The compiled regex should be usable after validation.
	require.True(t, cfg.Labels["label"].Matchers[0].regex.MatchString("something query something"))
	require.False(t, cfg.Labels["label"].Matchers[0].regex.MatchString("no match here"))
}

func TestConfig_Validate_InvalidRegexReturnsError(t *testing.T) {
	cfg := Config{
		Labels: map[string]Label{
			"label": {
				Matchers: []Matcher{
					{RegexStr: `[invalid`, Weight: 1},
				},
			},
		},
	}

	require.Error(t, cfg.Validate())
}
