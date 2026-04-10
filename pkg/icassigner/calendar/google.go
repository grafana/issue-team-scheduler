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

package calendar

import (
	"context"
	"fmt"
	"time"

	googlecalendar "google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

type GoogleConfigJSON string

func CheckGoogleAvailability(cfg GoogleConfigJSON, calendarName string, name string, now time.Time, unavailabilityLimit time.Duration) (bool, error) {
	opt := option.WithCredentialsJSON([]byte(cfg))
	calService, err := googlecalendar.NewService(context.Background(), opt)
	if err != nil {
		return true, fmt.Errorf("unable to get api access for %q, due %w", name, err)
	}

	response, err := calService.Freebusy.Query(&googlecalendar.FreeBusyRequest{
		TimeMin: time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
		TimeMax: time.Now().Add(4 * 24 * time.Hour).Format(time.RFC3339),
		Items: []*googlecalendar.FreeBusyRequestItem{
			{Id: calendarName},
		},
		TimeZone: "utc",
	}).Do()
	if err != nil {
		return true, fmt.Errorf("unable to get availaiblity from gcal, due %w", err)
	}

	calendar, ok := response.Calendars[calendarName]
	if !ok {
		return true, fmt.Errorf("unable to access calendar from %v, please ensure they shared their calendar with the service account", name)
	}
	if len(calendar.Errors) > 0 {
		return true, fmt.Errorf("unable to access calendar from %v, please ensure they shared their calendar with the service account. Internal error %q", name, calendar.Errors[0].Reason)
	}

	return checkGoogleBusySlots(calendar.Busy, now, unavailabilityLimit), nil
}

// checkGoogleBusySlots reports whether any of the freebusy slots blocks availability.
// Slots with unparseable RFC3339 times are skipped (same fail-open behaviour as the
// iCal path). Google Calendar always returns times in UTC, so UTC is used as the
// location for the look-ahead window calculation.
func checkGoogleBusySlots(slots []*googlecalendar.TimePeriod, now time.Time, unavailabilityLimit time.Duration) bool {
	checker := newIcalAvailabilityChecker(now, unavailabilityLimit, time.UTC)

	for _, e := range slots {
		start, err := time.Parse(time.RFC3339, e.Start)
		if err != nil {
			continue
		}

		end, err := time.Parse(time.RFC3339, e.End)
		if err != nil {
			continue
		}

		if checker.isEventBlockingAvailability(start, end) {
			return false
		}
	}

	return true
}
