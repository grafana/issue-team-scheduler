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
	"regexp"

	"gopkg.in/yaml.v2"
)

type Config struct {
	// Labels is a map of label names to matchers based on which the label gets a matching score.
	// The label with the highest score is applied to the issue.
	Labels map[string]Label `yaml:"labels,omitempty"`

	// RequireLabel is a list of labels, for the regex-labeler to run at least one of the specified labels must be present on the issue.
	RequireLabel []string `yaml:"requireLabel,omitempty"`
}

func (c *Config) Validate() error {
	for lKey, l := range c.Labels {
		if err := l.validate(); err != nil {
			return err
		}

		// the call to validate() can modify the label, so we need to put the result back.
		c.Labels[lKey] = l
	}
	return nil
}

type Label struct {
	// Matchers is a list of regular expressions to match against the title and body of an issue.
	Matchers []Matcher `yaml:"matchers,omitempty"`
}

func (l *Label) validate() error {
	for mIdx := range l.Matchers {
		// the call to validate() can modify the matcher, so we call it on the matcher in inside the slice.
		if err := l.Matchers[mIdx].validate(); err != nil {
			return err
		}
	}
	return nil
}

type Matcher struct {
	// RegexStr is the regular expression to match against the title and body of an issue.
	RegexStr string `yaml:"regex,omitempty"`

	// regex is the compiled version of RegexStr
	regex *regexp.Regexp `yaml:"-"`

	// Weight can be used to give this matcher more importance relative to other matchers.
	// Defaults to 1 if not specified.
	Weight int `yaml:"weight,omitempty"`
}

func (m *Matcher) validate() (err error) {
	if m.Weight == 0 {
		m.Weight = 1
	}
	m.regex, err = regexp.Compile(m.RegexStr)
	return err
}

func ParseConfig(cfg []byte) (cfgParsed Config, err error) {
	err = yaml.Unmarshal(cfg, &cfgParsed)
	return cfgParsed, err
}
