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
	"errors"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/google/go-github/github"
	githubaction "github.com/grafana/escalation-scheduler/pkg/github-action"
	"github.com/grafana/escalation-scheduler/pkg/icassigner/busyness"
	"github.com/grafana/escalation-scheduler/pkg/icassigner/calendar"
)

type Action struct {
	Client *github.Client
	Config Config
}

func (a *Action) Run(ctx context.Context, event *github.IssuesEvent, labelsInput string, dryRun bool) error {
	labels := convertLabels(event.Issue.Labels)
	if labelsInput != "" {
		labels = strings.Split(labelsInput, ",")
	}

	for _, i := range a.Config.IgnoredLabels {
		for _, l := range labels {
			if i == l {
				log.Printf("Label %q which marks an issue as to be ignored found. Stopping\n", i)
				return nil
			}
		}
	}

	// finds the team this issue is assigned to
	// TODO: Decide if we want to change this behavior if multiple teams match. We could create an adhoc bigger team and then simply distribute the escalations there
	teamMembers, teamName := findTeam(a.Config, labels)
	if len(teamMembers) == 0 {
		log.Print("No team is responsible for this issue. Stopping\n")
		return nil // no team is responsible for anything, so we just abort
	}

	// check if someone from the team is already assigned (skip in this case)
	if assigned, teamMember := isTeamMemberAssigned(teamMembers, event.Issue.Assignees); assigned {
		log.Printf("Found assignee %q which is member of the matched team %q. Stopping\n", teamMember, teamName)
		return nil
	}

	// Log the known team member names.
	log.Printf("Known team members: %q", strings.Join(memberNames(teamMembers), ", "))

	// 1. get busyness scores per team member
	// We calculate busyness first as this is usually cheaper than availability checks
	busynessPerTeamMember, err := a.calculateIssueBusynessPerTeamMember(ctx, time.Now(), teamMembers)
	if err != nil {
		return fmt.Errorf("unable to calculate team busyness, due %w", err)
	}

	// Log the busyness report.
	log.Printf("Team members by busyness: %q", busynessPerTeamMember.String())

	// 2. Iterate over team members by increasing busyness and check their availability
	var availableMembers []MemberConfig
	for _, b := range busynessPerTeamMember {
		for _, name := range b.Users {
			// find MemberConfig
			var member MemberConfig
			for _, m := range teamMembers {
				if m.Name == name {
					member = m
					break
				}
			}

			if member.Name == "" {
				continue
			}

			isAvailable, err := checkAvailability(member, a.Config.UnavailabilityLimit)
			if err != nil {
				log.Printf("Unable to fetch availability of %q, due %v", name, err)
			}

			if isAvailable {
				availableMembers = append(availableMembers, member)
			} else {
				log.Printf("Member %q is not available based on calendar", name)
			}
		}

		// if we found available team members we can stop
		if len(availableMembers) > 0 {
			break
		}
	}

	// In case no one is available we just consider everybody to be available
	if len(availableMembers) == 0 {
		log.Printf("Nobody seems to be available, hence we consider everybody to be available!")
		availableMembers = teamMembers
	}

	// Log the available team members.
	log.Printf("Available team members: %q", strings.Join(memberNames(availableMembers), ", "))

	// choose a member
	theChosenOne := availableMembers[rand.Intn(len(availableMembers))]
	log.Printf("Chose member %q.\n", theChosenOne.Name)

	// set output
	output := theChosenOne.Name
	if theChosenOne.Output != "" {
		output = theChosenOne.Output
	}

	githubaction.SetOutput("assignee", output)

	if dryRun {
		log.Print("Exiting because dry-run is enabled")
		return nil
	}

	if event.Repo == nil || event.Repo.Name == nil {
		log.Fatalf("Can't set any assignee as the repository or its name is missing, payload: %+v", event.Repo)
	}

	if event.Repo.Owner == nil || event.Repo.Owner.Login == nil {
		log.Fatalf("Can't set any assignee as the repository owner or its login name is missing, payload: %+v", event.Repo.Owner)
	}

	_, _, err = a.Client.Issues.AddAssignees(ctx, *event.Repo.Owner.Login, *event.Repo.Name, *event.Issue.Number, []string{theChosenOne.Name})

	return err
}

func memberNames(members []MemberConfig) (result []string) {
	for _, m := range members {
		result = append(result, m.Name)
	}
	return result
}

func (a *Action) calculateIssueBusynessPerTeamMember(ctx context.Context, now time.Time, members []MemberConfig) (busyness.Report, error) {
	team := make([]string, len(members))
	for i, m := range members {
		team[i] = m.Name
	}

	return busyness.CalculateBusynessForTeam(ctx, now, a.Client, a.Config.IgnoredLabels, team)
}

func checkAvailability(m MemberConfig, unavailabilityLimit time.Duration) (bool, error) {
	if m.GoogleCalendar != "" {
		cfg, err := GetGoogleConfig()
		if err != nil {
			return true, err
		}
		return calendar.CheckGoogleAvailability(cfg, m.GoogleCalendar, m.Name, time.Now(), unavailabilityLimit)
	}

	return calendar.CheckAvailability(m.IcalURL, m.Name, time.Now(), unavailabilityLimit)
}

func GetGoogleConfig() (calendar.GoogleConfigJSON, error) {
	clientSecret := githubaction.GetInputOrDefault("gcal-service-acount-key", "")

	if clientSecret == "" {
		return "", errors.New("can't fetch gcal availability due gcal_service_acount_key input not set")
	}
	return calendar.GoogleConfigJSON(clientSecret), nil
}

// findTeam finds the right team defined
func findTeam(cfg Config, labels []string) ([]MemberConfig, string) {
	labelMap := map[string]struct{}{}
	for _, l := range labels {
		labelMap[l] = struct{}{}
	}

	matchedTeams := []TeamConfig{}
	teamNames := []string{}
	for name, t := range cfg.Teams {
		match := false
		for _, l := range t.RequireLabel {
			if _, ok := labelMap[l]; ok {
				match = true
				break
			}
		}

		if !match {
			continue
		}

		matchedTeams = append(matchedTeams, t)
		teamNames = append(teamNames, name)
	}

	switch len(matchedTeams) {
	case 0:
		return nil, ""
	case 1:
		return matchedTeams[0].Members, teamNames[0]
	default:
		// if multiple teams match let's merge them
		// choose team randomly
		memberMap := map[string]MemberConfig{}
		for _, t := range matchedTeams {
			for _, m := range t.Members {
				if _, ok := memberMap[m.Name]; !ok {
					memberMap[m.Name] = m
				}
			}
		}

		members := make([]MemberConfig, 0, len(memberMap))
		for _, m := range memberMap {
			members = append(members, m)
		}

		name := fmt.Sprintf("Merged (%v)", strings.Join(teamNames, ", "))
		return members, name
	}
}

// convertLabels creates a string array with all label names from an array of github labels
func convertLabels(labels []github.Label) []string {
	labelStrings := make([]string, len(labels))
	for i, l := range labels {
		labelStrings[i] = *l.Name
	}

	return labelStrings
}

func isTeamMemberAssigned(teamMembers []MemberConfig, assignees []*github.User) (bool, string) {
	for _, m := range teamMembers {
		for _, a := range assignees {
			if strings.ToLower(a.GetLogin()) == strings.ToLower(m.Name) {
				return true, m.Name
			}
		}
	}

	return false, ""
}
