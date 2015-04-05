package types

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

var months map[string]time.Month
var days map[string]time.Weekday

// A Window type represents a known window of operation for a queue.  It is generated through the NLP parser as follows:
// window, err := parse(input_string)
//
// GetNextStartTime will return the correct starting point for the queue which the queueManager should use to initiate
// task execution.  When this start time is reached it is recommended that GetNextEndTime should be called to inform
// the system when to disable the queue again.
//
// All timezones are set to UTC unless specified in the window definition, e.g. "where timezone = GMT"
//
// BUG(kieranbroadfoot): If GetNextEndTime is called on a window which is "always on" and currently in the exception window
// the code will return a zero value.  It is recommended GetNextEndTime is called when the queueManager starts.
type Window struct {
	start_     time.Time
	end_       time.Time
	Start      string
	End        string
	Recurrence string
	OnDate     string
	Timezone   string
	AlwaysOn   bool
	AlwaysOff  bool
	Error      string
}

type date struct {
	day   int
	month time.Month
	year  int
}

func init() {
	months = map[string]time.Month{
		"jan":       time.January,
		"feb":       time.February,
		"mar":       time.March,
		"apr":       time.April,
		"may":       time.May,
		"jun":       time.June,
		"jul":       time.July,
		"aug":       time.August,
		"sep":       time.September,
		"oct":       time.October,
		"nov":       time.November,
		"dec":       time.December,
		"january":   time.January,
		"february":  time.February,
		"march":     time.March,
		"april":     time.April,
		"june":      time.June,
		"july":      time.July,
		"august":    time.August,
		"september": time.September,
		"october":   time.October,
		"november":  time.November,
		"december":  time.December,
	}
	days = map[string]time.Weekday{
		"sunday":    time.Sunday,
		"monday":    time.Monday,
		"tuesday":   time.Tuesday,
		"wednesday": time.Wednesday,
		"thursday":  time.Thursday,
		"friday":    time.Friday,
		"saturday":  time.Saturday,
		"sun":       time.Sunday,
		"mon":       time.Monday,
		"tue":       time.Tuesday,
		"wed":       time.Wednesday,
		"thu":       time.Thursday,
		"fri":       time.Friday,
		"sat":       time.Saturday,
	}
}

func (w *Window) GetNextStartTime() time.Time {
	return w.returnTime("start")
}

func (w *Window) GetNextEndTime() time.Time {
	return w.returnTime("end")
}

func (w *Window) returnTime(returntype string) time.Time {
	now := time.Now()
	if w.start_.IsZero() || w.end_.IsZero() || now.After(w.start_) {
		w.generateRecurringWindow()
	}
	if w.AlwaysOn == true && w.Recurrence != "" {
		// if the window is always open we should invert start/end
		if returntype == "start" {
			// hack: special case where we find ourselves in the window already.
			// the recurrence logic would have set start_ to now() because in
			// normal cases the queue should immediately start.  We need to invert
			// this because we do not want to be operating during the window
			now = now.Add(time.Second)
			if now.After(w.start_) && now.Before(w.end_) {
				// inside the exception window.  start when the exception window closes
				return w.end_
			} else {
				// else return now.  we are not in the window so should be running
				return time.Now()
			}
		} else {
			// See bug definition at the top of this file for details.
			now = now.Add(time.Second)
			if now.After(w.start_) && now.Before(w.end_) {
				// inside the exception window.  we cannot know the next end timestamp, return zero
				return time.Time{}
			} else {
				// not in the window, return start timestamp as exception window ending
				return w.start_
			}
		}
	} else {
		// standard case.
		if returntype == "start" {
			return w.start_
		} else {
			return w.end_
		}
	}
}

func (w *Window) generateRecurringWindow() {
	// return next available date based on recurring string
	now := time.Now()
	if w.AlwaysOn == true && w.Recurrence == "" {
		// simplest case.  we know this window is always open and there is no exceptions.
		// return now and far future
		w.start_ = now
		w.end_ = time.Date(2500, time.January, 0, 0, 0, 0, 0, time.UTC)
		return
	}

	if w.AlwaysOff == true {
		// a queue which is always off is easy.. set start and end to some far future date
		w.start_ = time.Date(2500, time.January, 0, 0, 0, 0, 0, time.UTC)
		w.end_ = time.Date(2501, time.January, 0, 0, 0, 0, 0, time.UTC)
	}

	if w.AlwaysOn == false && w.OnDate != "" {
		// the window has a very specific time/date combination set.  Simply determine if we are in/out of window and return
		w.start_, _ = generateTimeStamp(generatedateFromString(w.OnDate), w.Start, w.Timezone)
		w.end_, _ = generateTimeStamp(generatedateFromString(w.OnDate), w.End, w.Timezone)
		// code receiving output from this function should check the dates are future/past in order to determine if it should open the window of operation
		return
	}

	thisdate := date{day: now.Day(), month: now.Month(), year: now.Year()}

	currentTime := (now.Hour() * 60) + now.Minute()
	startTime := getStringTimeAsInt(w.Start)
	endTime := getStringTimeAsInt(w.End)

	startdate := thisdate
	enddate := thisdate

	elements := strings.Split(w.Recurrence, " ")
	if elements[0] == "day" {
		if endTime < startTime {
			// the end time is the next day
			enddate = addDaysTodate(enddate, 1)
		}
		// case: recurring daily event
		w.deriveWindowsForToday(currentTime, startTime, endTime, startdate, enddate)
		return
	}

	if isDay(elements[0]) {
		// this is a specific day request
		requestedDay := whichDay(elements[0])
		today := now.Weekday()
		if endTime < startTime {
			enddate = addDaysTodate(enddate, 1)
		}
		if today == requestedDay {
			if endTime < currentTime && startdate.day == enddate.day {
				// found a window which ended today prior to "now".  Fix up for next week
				w.start_, _ = generateTimeStamp(addDaysTodate(startdate, 7), w.Start, w.Timezone)
				w.end_, _ = generateTimeStamp(addDaysTodate(enddate, 7), w.End, w.Timezone)
			} else {
				w.deriveWindowsForToday(currentTime, startTime, endTime, startdate, enddate)
			}
		}
		if today < requestedDay {
			w.start_, _ = generateTimeStamp(addDaysTodate(startdate, int(requestedDay-today)), w.Start, w.Timezone)
			w.end_, _ = generateTimeStamp(addDaysTodate(enddate, int(requestedDay-today)), w.End, w.Timezone)
		}
		if today > requestedDay {
			w.start_, _ = generateTimeStamp(addDaysTodate(startdate, int((6-today)+requestedDay+1)), w.Start, w.Timezone)
			w.end_, _ = generateTimeStamp(addDaysTodate(enddate, int((6-today)+requestedDay+1)), w.End, w.Timezone)
		}
		return
	}

	if elements[0] == "weekday" || elements[0] == "weekend" {
		today := now.Weekday()
		if endTime < startTime {
			enddate = addDaysTodate(enddate, 1)
		}

		if elements[0] == "weekday" && !isTodayAWeekend(today) {
			w.deriveWindowsForToday(currentTime, startTime, endTime, startdate, enddate)
		} else if elements[0] == "weekday" && isTodayAWeekend(today) {
			// need to set for next available weekday which is monday.  so if today is saturday + 2, if sunday + 1
			daysToAdd := 2
			if today == time.Sunday {
				daysToAdd = 1
			}
			w.start_, _ = generateTimeStamp(addDaysTodate(startdate, daysToAdd), w.Start, w.Timezone)
			w.end_, _ = generateTimeStamp(addDaysTodate(enddate, daysToAdd), w.End, w.Timezone)
		} else if elements[0] == "weekend" && isTodayAWeekend(today) {
			w.deriveWindowsForToday(currentTime, startTime, endTime, startdate, enddate)
		} else {
			// today is a weekday and we want a weekend.  we are looking for saturday
			// so 6 - today = number of days to add
			w.start_, _ = generateTimeStamp(addDaysTodate(startdate, int(6-today)), w.Start, w.Timezone)
			w.end_, _ = generateTimeStamp(addDaysTodate(enddate, int(6-today)), w.End, w.Timezone)
		}
		return
	}

	if elements[1] == "month" || elements[1] == "monthly" {
		elementZero, err := strconv.Atoi(elements[0])
		if err == nil {
			startdate.day = elementZero
			enddate.day = elementZero
			if endTime < startTime {
				// the end time is the next day
				enddate = addDaysTodate(enddate, 1)
			}
			if elementZero < thisdate.day {
				// we need next month
				w.start_, _ = generateTimeStamp(addMonthTodate(startdate), w.Start, w.Timezone)
				w.end_, _ = generateTimeStamp(addMonthTodate(enddate), w.End, w.Timezone)
			}
			if elementZero > thisdate.day {
				// this month upcoming...
				w.start_, _ = generateTimeStamp(startdate, w.Start, w.Timezone)
				w.end_, _ = generateTimeStamp(enddate, w.End, w.Timezone)
			}
			if elementZero == thisdate.day {
				// its happening today
				w.deriveWindowsForToday(currentTime, startTime, endTime, startdate, enddate)
			}
		}
		return
	}

	if isMonth(elements[1]) {
		startdate.month = whichMonth(elements[1])
		enddate.month = whichMonth(elements[1])
		elementZero, err := strconv.Atoi(elements[0])
		if err == nil {
			startdate.day = elementZero
			enddate.day = elementZero
			if endTime < startTime {
				enddate = addDaysTodate(enddate, 1)
			}
			w.deriveWindowsForYear(thisdate, startdate, enddate, currentTime, startTime, endTime)
		}
		return
	}

	if elements[1] == "year" || elements[1] == "yearly" {
		for idx, item := range strings.Split(elements[0], "/") {
			item_, err := strconv.Atoi(item)
			if err == nil {
				if idx == 0 {
					startdate.day = item_
				} else if idx == 1 {
					startdate.month = time.Month(item_)
				}
			}
		}
		enddate = startdate
		if endTime < startTime {
			enddate = addDaysTodate(enddate, 1)
		}
		w.deriveWindowsForYear(thisdate, startdate, enddate, currentTime, startTime, endTime)
		return
	}
	return
}

func (w *Window) deriveWindowsForToday(currentTime int, startTime int, endTime int, startdate date, enddate date) {
	if startTime <= currentTime && endTime > currentTime {
		// currently in the window
		w.start_ = time.Now()
		w.end_, _ = generateTimeStamp(enddate, w.End, w.Timezone)
	} else if startTime > currentTime {
		// happens today in the future
		w.start_, _ = generateTimeStamp(startdate, w.Start, w.Timezone)
		w.end_, _ = generateTimeStamp(enddate, w.End, w.Timezone)
	} else if endTime < currentTime {
		// missed window for today, set to tomorrow
		w.start_, _ = generateTimeStamp(addDaysTodate(startdate, 1), w.Start, w.Timezone)
		w.end_, _ = generateTimeStamp(addDaysTodate(enddate, 1), w.End, w.Timezone)
	}
}

func (w *Window) deriveWindowsForYear(currentdate date, startdate date, enddate date, currentTime int, startTime int, endTime int) {
	if startdate.month < currentdate.month || startdate.month == currentdate.month && startdate.day < currentdate.day {
		// next year
		w.start_, _ = generateTimeStamp(addYearTodate(startdate), w.Start, w.Timezone)
		w.end_, _ = generateTimeStamp(addYearTodate(enddate), w.End, w.Timezone)
	} else if startdate.month > currentdate.month || startdate.month == currentdate.month && startdate.day > currentdate.day {
		w.start_, _ = generateTimeStamp(startdate, w.Start, w.Timezone)
		w.end_, _ = generateTimeStamp(enddate, w.End, w.Timezone)
	} else if startdate.month == currentdate.month && startdate.day == currentdate.day {
		// today
		w.deriveWindowsForToday(currentTime, startTime, endTime, startdate, enddate)
	}
}

func getStringTimeAsInt(input string) int {
	// calculate number of minutes since midnight for the given input
	elements := strings.Split(input, ":")
	hours, herr := strconv.Atoi(elements[0])
	minutes, merr := strconv.Atoi(elements[1])
	if herr == nil && merr == nil {
		return (hours * 60) + minutes
	} else {
		return -1
	}
}

func addDaysTodate(inputdate date, addition int) date {
	// return a date which is X days in the future.  expect addition < 7
	returndate := date{day: inputdate.day, month: inputdate.month, year: inputdate.year}
	numberOfDaysInMonth := time.Date(inputdate.year, inputdate.month+1, 0, 0, 0, 0, 0, time.UTC).Day()
	if inputdate.day+addition <= numberOfDaysInMonth {
		// easiest case.
		returndate.day = inputdate.day + addition
		return returndate
	} else {
		// we are increasing the month (and potentially the year)
		returndate.day = inputdate.day + addition - numberOfDaysInMonth
		if inputdate.month == 12 {
			returndate.year = returndate.year + addition
			returndate.month = time.January
		} else {
			returndate.month++
		}
		return returndate
	}
}

func addMonthTodate(inputdate date) date {
	returndate := date{day: inputdate.day, month: inputdate.month, year: inputdate.year}
	if returndate.month == 12 {
		returndate.month = time.January
		returndate.year = returndate.year + 1
	} else {
		returndate.month++
	}
	return returndate
}

func addYearTodate(inputdate date) date {
	returndate := date{day: inputdate.day, month: inputdate.month, year: inputdate.year + 1}
	return returndate
}

func isTodayAWeekend(day time.Weekday) bool {
	if day == time.Saturday || day == time.Sunday {
		return true
	}
	return false
}

func isDay(input string) bool {
	for key, _ := range days {
		if input == key {
			return true
		}
	}
	return false
}

func whichDay(input string) time.Weekday {
	for key, value := range days {
		if input == key {
			return value
		}
	}
	return time.Sunday
}

func isMonth(input string) bool {
	for key, _ := range months {
		if input == key {
			return true
		}
	}
	return false
}

func whichMonth(input string) time.Month {
	for key, value := range months {
		if input == key {
			return value
		}
	}
	return time.January
}

func generatedateFromString(input string) date {
	elements := strings.Split(input, "/")
	day, _ := strconv.Atoi(elements[0])
	month, _ := strconv.Atoi(elements[1])
	year, _ := strconv.Atoi(elements[2])
	return date{day: day, month: time.Month(month), year: year}
}

func generateTimeStamp(date date, time_ string, timezone string) (time.Time, error) {
	if len(timezone) == 0 {
		timezone = "UTC"
	}
	format := "02/01/2006 15:04 MST"
	input := fmt.Sprintf("%02d/%02d/%04d %s %s", date.day, int(date.month), date.year, time_, timezone)
	newTime, err := time.Parse(format, input)
	if err == nil {
		return newTime, nil
	}
	return time.Now(), err
}
