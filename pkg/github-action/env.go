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

package githubaction

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/google/go-github/github"
)

func LoadContext() (*github.IssuesEvent, error) {
	var ctx *github.IssuesEvent

	eventName := os.Getenv("GITHUB_EVENT_NAME")
	if eventName == "" {
		return ctx, errors.New("missing env var GITHUB_EVENT_NAME")
	}

	eventFile := os.Getenv("GITHUB_EVENT_PATH")
	if eventFile == "" {
		return ctx, errors.New("missing env var GITHUB_EVENT_PATH")
	}

	rawContext, err := os.ReadFile(eventFile)
	if err != nil {
		return ctx, fmt.Errorf("error when reading event file: %w", err)
	}

	githubCtx, err := github.ParseWebHook(eventName, rawContext)
	if err != nil {
		return ctx, fmt.Errorf("error parsing github event: %w", err)
	}

	ctx, ok := githubCtx.(*github.IssuesEvent)
	if !ok {
		return ctx, errors.New("event is not an issue event")
	}

	return ctx, err
}

func Repository() (string, string, string, error) {
	e := os.Getenv("GITHUB_REPOSITORY")
	if e == "" {
		return "", "", "", errors.New("missing env var GITHUB_REPOSITORY")
	}

	parts := strings.Split(e, "/")
	if len(parts) != 2 {
		return "", "", "", fmt.Errorf("expected 2 parts (owner + repo) in GITHUB_REPOSITORY, but got %v", len(parts))
	}

	sha := os.Getenv("GITHUB_SHA")
	if sha == "" {
		return "", "", "", errors.New("missing env var GITHUB_SHA")
	}

	return parts[0], parts[1], sha, nil
}
