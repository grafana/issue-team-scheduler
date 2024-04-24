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

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"

	"github.com/google/go-github/github"
	githubaction "github.com/grafana/escalation-scheduler/pkg/github-action"
	"github.com/grafana/escalation-scheduler/pkg/labeler"
)

func main() {
	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))

	actionCtx, err := githubaction.LoadContext()
	if err != nil {
		level.Error(logger).Log("msg", "unable to load github context", "err", err)
		return
	}

	if actionCtx.Issue == nil {
		level.Error(logger).Log("msg", "can not be used without an issue")
		return
	}

	if actionCtx.Issue.GetState() != "open" {
		level.Error(logger).Log("msg", "only works on currently open issues", "currentState", actionCtx.Issue.GetState())
		return
	}

	owner, repo, sha, err := githubaction.Repository()
	if err != nil {
		level.Error(logger).Log("msg", "unable to identify current github repo", "err", err)
		return
	}

	cfgPath := githubaction.GetInputOrDefault("cfg-path", "./github/squad-assignment.yaml")

	gh, err := githubaction.NewGithubClientFromEnv()
	if err != nil {
		level.Error(logger).Log("msg", "failed to get github client", "err", err)
		return
	}

	cfgContent, err := fetchConfig(gh, owner, repo, sha, cfgPath)
	if err != nil {
		level.Error(logger).Log("msg", "unable to get config", "err", err)
		return
	}

	cfg, err := labeler.ParseConfig(cfgContent)
	if err != nil {
		level.Error(logger).Log("msg", "error when parsing config", "err", err)
		return
	}

	err = cfg.Validate()
	if err != nil {
		level.Error(logger).Log("msg", "error when validating config", "err", err)
		return
	}

	eventName := os.Getenv("GITHUB_EVENT_NAME")
	if eventName == "" {
		level.Error(logger).Log("msg", "missing env var GITHUB_EVENT_NAME")
		return
	}

	eventFile := os.Getenv("GITHUB_EVENT_PATH")
	if eventFile == "" {
		level.Error(logger).Log("msg", "missing env var GITHUB_EVENT_PATH")
		return
	}

	rawContext, err := os.ReadFile(eventFile)
	if err != nil {
		level.Error(logger).Log("msg", "error when reading event file", "err", err)
		return
	}

	githubCtx, err := github.ParseWebHook(eventName, rawContext)
	if err != nil {
		level.Error(logger).Log("msg", "error parsing github web hook", "err", err)
		return
	}

	issuesEvent, ok := githubCtx.(*github.IssuesEvent)
	if !ok {
		level.Error(logger).Log("msg", "event is not an issue event")
		return
	}

	l := labeler.NewLabeler(cfg, gh, logger)
	issue := issuesEvent.GetIssue()
	err = l.Run(issue)
	if err != nil {
		level.Error(logger).Log("msg", "failed to assign label", "err", err)
		return
	}
}

func fetchConfig(client *github.Client, owner, repo, ref, path string) ([]byte, error) {
	rawContent, _, _, err := client.Repositories.GetContents(context.Background(), owner, repo, path, &github.RepositoryContentGetOptions{
		Ref: ref,
	})

	if err != nil {
		return nil, fmt.Errorf("unable to retrieve config, due %w", err)
	}

	content, err := rawContent.GetContent()
	if err != nil {
		return nil, fmt.Errorf("unable to load config, due %w", err)
	}

	return []byte(content), nil
}
