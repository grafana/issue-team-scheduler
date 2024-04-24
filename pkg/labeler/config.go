package labeler

import (
	"regexp"

	"gopkg.in/yaml.v2"
)

type Config struct {
	// Labels is a map of label names to matchers based on which the label gets a matching score.
	// The label with the highest score is applied to the issue.
	Labels map[string]Label `yaml:"labels,omitempty"`

	// RequiredLabels is a list of labels that must be present on the issue for any other labels to be applied.
	RequiredLabels []string `yaml:"required_labels,omitempty"`
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
