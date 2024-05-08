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
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "time/tzdata"
)

func TestIsEventBlockingAvailability(t *testing.T) {
	now := time.Now()

	utc, err := time.LoadLocation("UTC")
	if err != nil {
		t.Fatal("Error during timezone utc loading:", err)
	}
	usa, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal("Error during timezone usa loading:", err)
	}

	australia, err := time.LoadLocation("Australia/Melbourne")
	if err != nil {
		t.Fatal("Error during timezone melbourne loading:", err)
	}

	testCases := []struct {
		name string

		now        time.Time
		start, end time.Time
		location   *time.Location

		expectedResult bool
	}{
		{
			name:     "quick 20min sync during the day",
			now:      now,
			start:    now.Add(2 * time.Hour),
			end:      now.Add(2*time.Hour + 20*time.Minute),
			location: time.Local,

			expectedResult: false,
		},
		{
			name:     "4hr appointment should block",
			now:      now,
			start:    now.Add(1 * time.Hour),
			end:      now.Add(7 * time.Hour),
			location: time.Local,

			expectedResult: true,
		},
		{
			name:     "PTO the whole day, but more than 12hr in advance",
			location: usa,
			start:    time.Date(2023, time.August, 23, 19, 0, 0, 0, utc),
			end:      time.Date(2023, time.August, 24, 19, 0, 0, 0, utc),
			now:      time.Date(2023, time.August, 23, 6, 30, 0, 0, utc),

			expectedResult: false,
		},
		{
			name:     "PTO the whole day, but less than 12hr in advance",
			location: usa,
			start:    time.Date(2023, time.August, 23, 19, 0, 0, 0, utc),
			end:      time.Date(2023, time.August, 24, 19, 0, 0, 0, utc),
			now:      time.Date(2023, time.August, 23, 14, 0, 0, 0, utc),

			expectedResult: true,
		},
		{
			name:     "during pto",
			location: usa,
			start:    time.Date(2023, time.August, 23, 19, 0, 0, 0, utc),
			end:      time.Date(2023, time.August, 24, 19, 0, 0, 0, utc),
			now:      time.Date(2023, time.August, 24, 15, 0, 0, 0, utc),

			expectedResult: true,
		},
		{
			name:     "after pto",
			location: usa,
			start:    time.Date(2023, time.August, 23, 19, 0, 0, 0, utc),
			end:      time.Date(2023, time.August, 24, 19, 0, 0, 0, utc),
			now:      time.Date(2023, time.August, 24, 19, 10, 0, 0, utc),

			expectedResult: false,
		},
		{
			name:     "Friday morning before PTO",
			location: australia,
			start:    time.Date(2023, time.August, 27, 13, 0, 0, 0, utc),
			end:      time.Date(2023, time.September, 01, 13, 0, 0, 0, utc),

			now:            time.Date(2023, time.August, 24, 21, 0, 0, 0, utc),
			expectedResult: false,
		},
		{
			name:     "Friday evening before PTO",
			location: australia,
			start:    time.Date(2023, time.August, 27, 13, 0, 0, 0, utc),
			end:      time.Date(2023, time.September, 01, 13, 0, 0, 0, utc),

			now:            time.Date(2023, time.August, 25, 10, 00, 0, 0, utc),
			expectedResult: true,
		},
		{
			name:     "Saturday morning before PTO",
			location: australia,
			start:    time.Date(2023, time.August, 27, 13, 0, 0, 0, utc),
			end:      time.Date(2023, time.September, 01, 13, 0, 0, 0, utc),

			now:            time.Date(2023, time.August, 25, 22, 00, 0, 0, utc),
			expectedResult: true,
		},
		{
			name:     "Saturday evening before PTO",
			location: australia,
			start:    time.Date(2023, time.August, 27, 13, 0, 0, 0, utc),
			end:      time.Date(2023, time.September, 01, 13, 0, 0, 0, utc),

			now:            time.Date(2023, time.August, 25, 26, 9, 0, 0, utc),
			expectedResult: true,
		},
		{
			name:     "Sunday morning before PTO",
			location: australia,
			start:    time.Date(2023, time.August, 27, 13, 0, 0, 0, utc),
			end:      time.Date(2023, time.September, 01, 13, 0, 0, 0, utc),

			now:            time.Date(2023, time.August, 26, 22, 00, 0, 0, utc),
			expectedResult: true,
		},
		{
			name:     "Sunday evening before PTO",
			location: australia,
			start:    time.Date(2023, time.August, 27, 13, 0, 0, 0, utc),
			end:      time.Date(2023, time.September, 01, 13, 0, 0, 0, utc),

			now:            time.Date(2023, time.August, 25, 27, 11, 0, 0, utc),
			expectedResult: true,
		},
		{
			name:     "during PTO",
			location: australia,
			start:    time.Date(2023, time.August, 27, 13, 0, 0, 0, utc),
			end:      time.Date(2023, time.September, 01, 13, 0, 0, 0, utc),

			now:            time.Date(2023, time.August, 29, 12, 25, 0, 0, utc),
			expectedResult: true,
		},
		{
			name:     "during weekend after PTO",
			location: australia,
			start:    time.Date(2023, time.August, 27, 13, 0, 0, 0, utc),
			end:      time.Date(2023, time.September, 01, 13, 0, 0, 0, utc),

			now:            time.Date(2023, time.September, 02, 01, 00, 0, 0, utc),
			expectedResult: false,
		},
		{
			name:     "Monday morning after PTO",
			location: australia,
			start:    time.Date(2023, time.August, 27, 13, 0, 0, 0, utc),
			end:      time.Date(2023, time.September, 01, 13, 0, 0, 0, utc),

			now:            time.Date(2023, time.September, 03, 23, 00, 0, 0, utc),
			expectedResult: false,
		},
	}

	for _, testcase := range testCases {
		t.Run(testcase.name, func(t *testing.T) {
			availabilityChecker := newIcalAvailabilityChecker(testcase.now, 6*time.Hour, testcase.location)
			res := availabilityChecker.isEventBlockingAvailability(testcase.start, testcase.end)
			if res != testcase.expectedResult {
				t.Errorf("Expected isEventBlockingAvailability to be %v, but got %v for event between %q and %q (tz=%v)", testcase.expectedResult, res, testcase.start, testcase.end, testcase.location.String())
			}
		})
	}

}

func TestCheckAvailability(t *testing.T) {
	ical := `BEGIN:VCALENDAR
PRODID:-//Google Inc//Google Calendar 70.9054//EN
VERSION:2.0
CALSCALE:GREGORIAN
METHOD:PUBLISH
X-WR-CALNAME:Test
X-WR-TIMEZONE:America/Los_Angeles
BEGIN:VEVENT
DTSTART:20231207T150000Z
DTEND:20231208T002000Z
DTSTAMP:20231207T151232Z
UID:abcdefghijklmonpqrstuvwxyz@google.com
CREATED:20231207T144208Z
LAST-MODIFIED:20231207T144253Z
SEQUENCE:0
STATUS:CONFIRMED
SUMMARY:asdasdef
TRANSP:OPAQUE
END:VEVENT
END:VCALENDAR`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, ical)
	}))
	defer ts.Close()

	loc, _ := time.LoadLocation("UTC")
	now := time.Date(2023, time.December, 07, 16, 0, 0, 0, loc)

	r, err := CheckAvailability(ts.URL, "tester", now, DefaultUnavailabilityLimit)

	if err != nil {
		t.Errorf("No error expected during basic ical check, but got %v", err)
	}

	if r {
		t.Errorf("Expected CheckAvailability to return unavailble, but got %v", r)
	}
}
