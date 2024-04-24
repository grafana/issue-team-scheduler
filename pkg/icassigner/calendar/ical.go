package calendar

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/emersion/go-ical"
)

const UnavailabilityLimit = 6 * time.Hour // 6hr

func isEventBlockingAvailability(now time.Time, start, end time.Time, loc *time.Location) bool {
	// if event is shorter than unavailabilityLimit, skip it
	if end.Sub(start) < UnavailabilityLimit {
		return false
	}

	// if the end of this date is already before the current date, skip it
	if end.Before(now) {
		return false
	}

	// At this point we know that
	// - the event is longer than UnavailabilityLimit
	// - it didn't happen in the past
	//
	// Now we need to check if that event starts in the next 12 business hours
	lookAheadTime := 12 * time.Hour
	localDate := now.In(loc)

	switch localDate.Weekday() {
	case time.Friday:
		// On Fridays we add two days to check if Monday is free if issue comes in during the second half of the day.
		lookAheadTime += 2 * 24 * time.Hour
	case time.Saturday:
		// On Saturdays we add a day and a half to check if Monday is free.
		lookAheadTime += 1.5 * 24 * time.Hour
	case time.Sunday:
		// On Sunday we add half a day to check if Monday is free.
		lookAheadTime += 0.5 * 24 * time.Hour
	}

	// if start is beyond future lookup it doesn't block availability
	return !start.After(localDate.Add(lookAheadTime))
}

func parseStartEnd(e ical.Event, loc *time.Location) (time.Time, time.Time, error) {
	start, err := e.DateTimeStart(loc)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("unable to parse date time start of event due %w", err)
	}

	end, err := e.DateTimeEnd(loc)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("unable to parse date time end of event due %w", err)
	}

	return start, end, nil
}

func checkEvents(events []ical.Event, name string, now time.Time, loc *time.Location) (bool, error) {
	for _, event := range events {
		if prop := event.Props.Get(ical.PropTransparency); prop != nil && prop.Value == "TRANSPARENT" {
			continue
		}

		start, end, err := parseStartEnd(event, loc)
		if err != nil {
			log.Printf("Unable to parse start/end of an event, due %v\n", err)
			continue
		}

		// check original occurence
		if isEventBlockingAvailability(now, start, end, loc) {
			log.Printf("calendar.isAvailableOn: person %q in %q is unavailable due to event from %q to %q\n", name, loc.String(), start, end)
			return false, nil
		}

		// if event has no reoccurence rule we are done;
		reoccurences, err := event.RecurrenceSet(loc)
		if err != nil || reoccurences == nil {
			continue
		}

		completeDuration := end.Sub(start)
		startOfReoccurences := now.Add(-2 * completeDuration)
		endOfReoccurences := now.Add(2 * completeDuration)

		occurences := reoccurences.Between(startOfReoccurences, endOfReoccurences, true)
		for _, o := range occurences {
			start := o
			end := o.Add(completeDuration)

			if isEventBlockingAvailability(now, start, end, loc) {
				log.Printf(`calendar.isAvailableOn: person %q is unavailable due to event from %q to %q`, name, start, end)
				return false, nil
			}
		}
	}

	return true, nil
}

func CheckAvailability(icalUrl string, name string, now time.Time) (bool, error) {
	resp, err := http.Get(icalUrl)
	if err != nil {
		return true, fmt.Errorf("unable to download ical file, due %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return true, fmt.Errorf("unable to download ical file, due non 200 status code %v", resp.StatusCode)
	}

	cal, err := ical.NewDecoder(resp.Body).Decode()
	if err != nil {
		return true, fmt.Errorf("unable to parse ical, due %w", err)
	}

	var loc *time.Location
	if tzString := cal.Props.Get("X-WR-TIMEZONE"); tzString != nil {
		loc, err = time.LoadLocation(tzString.Value)
		if err != nil {
			log.Printf("Unable to parse timezone %q, due %v", tzString.Value, err)
		}
	}

	return checkEvents(cal.Events(), name, now, loc)
}
