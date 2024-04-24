package calendar

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

type GoogleConfigJSON string

func CheckGoogleAvailability(cfg GoogleConfigJSON, calendarName string, name string, now time.Time) (bool, error) {
	opt := option.WithCredentialsJSON([]byte(cfg))
	calService, err := calendar.NewService(context.Background(), opt)
	if err != nil {
		return true, fmt.Errorf("unable to get api access for %q, due %w", name, err)
	}

	response, err := calService.Freebusy.Query(&calendar.FreeBusyRequest{
		TimeMin: time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
		TimeMax: time.Now().Add(4 * 24 * time.Hour).Format(time.RFC3339),
		Items: []*calendar.FreeBusyRequestItem{
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

	// check all events
	for _, e := range calendar.Busy {
		start, err := time.Parse(time.RFC3339, e.Start)
		if err != nil { // in case of an error -> skip event
			continue
		}

		end, err := time.Parse(time.RFC3339, e.End)
		if err != nil { // in case of an error -> skip event
			continue
		}

		if isEventBlockingAvailability(now, start, end, time.UTC) {
			return false, nil
		}
	}

	return true, nil
}
