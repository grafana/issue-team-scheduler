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
	"testing"
	"time"

	googlecalendar "google.golang.org/api/calendar/v3"

	"github.com/stretchr/testify/require"
)

// slot is a helper to build a *googlecalendar.TimePeriod from plain UTC time strings.
func slot(start, end string) *googlecalendar.TimePeriod {
	return &googlecalendar.TimePeriod{Start: start, End: end}
}

func TestCheckGoogleBusySlots_NoSlots(t *testing.T) {
	now := time.Date(2024, time.January, 11, 8, 0, 0, 0, time.UTC)
	available := checkGoogleBusySlots(nil, now, DefaultUnavailabilityLimit)
	require.True(t, available, "expected available when there are no busy slots")
}

func TestCheckGoogleBusySlots_ShortSlotDoesNotBlock(t *testing.T) {
	// A slot shorter than the unavailability limit must not block.
	now := time.Date(2024, time.January, 11, 8, 0, 0, 0, time.UTC)
	slots := []*googlecalendar.TimePeriod{
		slot("2024-01-11T09:00:00Z", "2024-01-11T09:30:00Z"), // 30 min
	}
	available := checkGoogleBusySlots(slots, now, DefaultUnavailabilityLimit)
	require.True(t, available, "expected available: slot is shorter than unavailability limit")
}

func TestCheckGoogleBusySlots_LongSlotWithinLookaheadBlocks(t *testing.T) {
	// A slot longer than the limit that starts within the 12h lookahead must block.
	now := time.Date(2024, time.January, 11, 8, 0, 0, 0, time.UTC)
	slots := []*googlecalendar.TimePeriod{
		slot("2024-01-11T09:00:00Z", "2024-01-11T17:00:00Z"), // 8h, starts in 1h
	}
	available := checkGoogleBusySlots(slots, now, DefaultUnavailabilityLimit)
	require.False(t, available, "expected unavailable: 8h slot within 12h lookahead")
}

func TestCheckGoogleBusySlots_SlotInPastDoesNotBlock(t *testing.T) {
	now := time.Date(2024, time.January, 11, 18, 0, 0, 0, time.UTC)
	slots := []*googlecalendar.TimePeriod{
		slot("2024-01-11T09:00:00Z", "2024-01-11T17:00:00Z"), // 8h, already over
	}
	available := checkGoogleBusySlots(slots, now, DefaultUnavailabilityLimit)
	require.True(t, available, "expected available: slot ended before now")
}

func TestCheckGoogleBusySlots_MultipleSlots_OneBlocking(t *testing.T) {
	// A non-blocking slot must not prevent a blocking slot from being detected.
	now := time.Date(2024, time.January, 11, 8, 0, 0, 0, time.UTC)
	slots := []*googlecalendar.TimePeriod{
		slot("2024-01-11T08:30:00Z", "2024-01-11T09:00:00Z"), // 30 min — non-blocking
		slot("2024-01-11T09:00:00Z", "2024-01-11T17:00:00Z"), // 8h — blocking
	}
	available := checkGoogleBusySlots(slots, now, DefaultUnavailabilityLimit)
	require.False(t, available, "expected unavailable: second slot is blocking")
}

func TestCheckGoogleBusySlots_UnparseableStartIsSkipped(t *testing.T) {
	// A slot with an unparseable start time must be skipped (fail-open).
	// If it is the only slot, the person is still considered available.
	now := time.Date(2024, time.January, 11, 8, 0, 0, 0, time.UTC)
	slots := []*googlecalendar.TimePeriod{
		slot("not-a-date", "2024-01-11T17:00:00Z"),
	}
	available := checkGoogleBusySlots(slots, now, DefaultUnavailabilityLimit)
	require.True(t, available, "expected available: slot with unparseable start is skipped")
}

func TestCheckGoogleBusySlots_UnparseableEndIsSkipped(t *testing.T) {
	// A slot with an unparseable end time must be skipped (fail-open).
	now := time.Date(2024, time.January, 11, 8, 0, 0, 0, time.UTC)
	slots := []*googlecalendar.TimePeriod{
		slot("2024-01-11T09:00:00Z", "not-a-date"),
	}
	available := checkGoogleBusySlots(slots, now, DefaultUnavailabilityLimit)
	require.True(t, available, "expected available: slot with unparseable end is skipped")
}

func TestCheckGoogleBusySlots_UnparseableSlotSkipped_OtherSlotStillBlocks(t *testing.T) {
	// An unparseable slot is skipped, but a valid blocking slot after it still applies.
	now := time.Date(2024, time.January, 11, 8, 0, 0, 0, time.UTC)
	slots := []*googlecalendar.TimePeriod{
		slot("not-a-date", "2024-01-11T17:00:00Z"),           // skipped
		slot("2024-01-11T09:00:00Z", "2024-01-11T17:00:00Z"), // 8h — blocking
	}
	available := checkGoogleBusySlots(slots, now, DefaultUnavailabilityLimit)
	require.False(t, available, "expected unavailable: valid blocking slot present after unparseable one")
}
