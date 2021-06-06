package rrule

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

const (
	rfc5545Date          = "20060102"
	rfc5545WithOffset    = "20060102T150405Z0700"
	rfc5545WithoutOffset = "20060102T150405"
)

// parseTime parses the time. the boolean is true if the time was in "local" (aka "floating")
// time, and thus the defautlLoc was used.
func parseTime(str string, defaultLoc *time.Location) (time.Time, bool, error) {
	//        DTSTART;TZID=America/New_York:19970902T090000

	var t time.Time

	if defaultLoc == nil {
		defaultLoc = time.UTC
	}
	loc := defaultLoc
	tzidFound := false

	if idBeg := strings.Index(str, ";TZID="); idBeg >= 0 {
		locBeg := idBeg + 6
		locEnd := locBeg + strings.Index(str[locBeg:], ":")
		if locEnd < 0 {
			return t, false, errors.New("no end to TZID")
		}

		var err error
		loc, err = LoadLocation(str[locBeg:locEnd])
		if err != nil {
			return t, false, err
		}

		tzidFound = true
		str = str[locEnd+1:]
	} else {
		colonIdx := strings.IndexAny(str, ":=")
		str = str[colonIdx+1:]
	}

	offsetFound := true

	t, err := time.ParseInLocation(rfc5545WithOffset, str, loc)
	if err != nil {
		offsetFound = false
		t, err = time.ParseInLocation(rfc5545WithoutOffset, str, loc)
		if err != nil {
			t, err = time.ParseInLocation(rfc5545Date, str, loc)
		}
	}

	// From RFC 5545:
	//
	//     If, based on the definition of the referenced time zone, the local
	//     time described occurs more than once (when changing from daylight
	//     to standard time), the DATE-TIME value refers to the first
	//     occurrence of the referenced time.  Thus, TZID=America/
	//     New_York:20071104T013000 indicates November 4, 2007 at 1:30 A.M.
	//     EDT (UTC-04:00).  If the local time described does not occur (when
	//     changing from standard to daylight time), the DATE-TIME value is
	//     interpreted using the UTC offset before the gap in local times.
	//     Thus, TZID=America/New_York:20070311T023000 indicates March 11,
	//     2007 at 3:30 A.M. EDT (UTC-04:00), one hour after 1:30 A.M. EST
	//     (UTC-05:00).
	//
	// However, Go's time.ParseInLocation makes no guarantee about how it
	// behaves relative to "fall-back" repetition of an hour in DST
	// transitions. (time.Date explicitly documents the same concept is
	// undefined.) Therefore, here, we normalize according to the spec by
	// trying to remove an hour and see if the local time is the same, and
	// if so, we keep that difference. Otherwise, if the original string was
	// in the 2am range, but the parsed time is less than 2 o'clock, advance an hour.
	if tMinusHour := t.Add(-1 * time.Hour); t.Hour() == tMinusHour.Hour() {
		t = tMinusHour
	} else if twoAMRegex.MatchString(str) {
		t = t.Add(1 * time.Hour)
	}

	return t, !(tzidFound || offsetFound), err
}

var twoAMRegex = regexp.MustCompile("T02[0-9]{4}(Z|[0-9]{4})?$")

func formatTime(prefix string, t time.Time, floatingLocation bool) string {
	if floatingLocation {
		return fmt.Sprintf("%s:%s", prefix, t.Format(rfc5545WithoutOffset))
	}

	if t.Location() == time.UTC {
		return fmt.Sprintf("%s:%sZ", prefix, t.Format(rfc5545WithoutOffset))
	}

	return fmt.Sprintf("%s;TZID=%s:%s", prefix, t.Location(), t.Format(rfc5545WithoutOffset))
}
