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
	"log"

	githubaction "github.com/grafana/escalation-scheduler/pkg/github-action"
	"github.com/grafana/escalation-scheduler/pkg/icassigner"
)

func main() {
	actionCtx, err := githubaction.LoadContext()
	if err != nil {
		log.Fatalf("Unable to load github context, due: %v", err)
	}

	if actionCtx.Issue == nil {
		log.Fatal("Can not be used without an issue")
	}

	if actionCtx.Issue.State == nil || *actionCtx.Issue.State != "open" {
		log.Fatalf("Only works on currently open issues, but found %q. Stopping...", *actionCtx.Issue.State)
	}

	owner, repo, sha, err := githubaction.Repository()
	if err != nil {
		log.Fatalf("Unable to identify current github repo, due: %v", err)
	}

	cfgPath := githubaction.GetInputOrDefault("cfg-path", "./github/escalation-assignment.yaml")

	ctx := context.Background()

	client, err := githubaction.NewGithubClientFromEnv()
	if err != nil {
		log.Fatalf("Unable to create github client: %v", err)
	}

	cfgReader, err := icassigner.FetchConfig(ctx, client, owner, repo, sha, cfgPath)
	if err != nil {
		log.Fatalf("Unable to get config: %v", err)
	}

	cfg, err := icassigner.ParseConfig(cfgReader)
	if err != nil {
		log.Fatalf("Unable to parse config: %v", err)
	}

	if actionCtx.Issue == nil {
		log.Fatalf("Unable to run without an issue set")
	}

	if actionCtx.Issue.Number == nil {
		log.Fatalf("No issue number is set")
	}

	dryRun := githubaction.GetInputOrDefault("dry-run", "true") != "false"

	labelsList := githubaction.GetInputOrDefault("labels", "")

	action := &icassigner.Action{
		Client: client,
		Config: cfg,
	}

	err = action.Run(ctx, actionCtx, labelsList, dryRun)
	if err != nil {
		log.Fatalf("Unable to run action: %v", err)
	}
}
