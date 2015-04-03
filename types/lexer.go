package types

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var mapper map[string]itemType
var RE_TIME *regexp.Regexp
var RE_CALENDAR *regexp.Regexp

func init() {
	mapper = map[string]itemType{
		"-":         itemTimeRange,
		"to":        itemTimeRange,
		"where":     itemWhere,
		"and":       itemAnd,
		"pm":        itemTimeModifier,
		"am":        itemTimeModifier,
		"every":     itemRecurrence,
		"any":       itemAnyAlways,
		"always":    itemAnyAlways,
		"never":     itemNever,
		"time":      itemAnyAlwaysClarification,
		"except":    itemException,
		"on":        itemOn,
		"=":         itemEquals,
		"timezone":  itemTimeZoneParam,
		"of":        itemClarification,
		"the":       itemClarification,
		"sunday":    itemDay,
		"monday":    itemDay,
		"tuesday":   itemDay,
		"wednesday": itemDay,
		"thursday":  itemDay,
		"friday":    itemDay,
		"saturday":  itemDay,
		"sun":       itemDay,
		"mon":       itemDay,
		"tue":       itemDay,
		"wed":       itemDay,
		"thu":       itemDay,
		"fri":       itemDay,
		"sat":       itemDay,
		"weekday":   itemDay,
		"weekend":   itemDay,
		"jan":       itemMonth,
		"feb":       itemMonth,
		"mar":       itemMonth,
		"apr":       itemMonth,
		"may":       itemMonth,
		"jun":       itemMonth,
		"jul":       itemMonth,
		"aug":       itemMonth,
		"sep":       itemMonth,
		"oct":       itemMonth,
		"nov":       itemMonth,
		"dec":       itemMonth,
		"january":   itemMonth,
		"february":  itemMonth,
		"march":     itemMonth,
		"april":     itemMonth,
		"june":      itemMonth,
		"july":      itemMonth,
		"august":    itemMonth,
		"september": itemMonth,
		"october":   itemMonth,
		"november":  itemMonth,
		"december":  itemMonth,
		"day":       itemRecurringByDayMonthYear,
		"month":     itemRecurringByDayMonthYear,
		"year":      itemRecurringByDayMonthYear,
		"monthly":   itemRecurringByDayMonthYear,
		"yearly":    itemRecurringByDayMonthYear,
	}
	RE_TIME = regexp.MustCompile("(?P<hour>\\d{1,2}):?(?P<minute>\\d{2})?\\s?(?P<mod>am|pm)?")
	RE_CALENDAR = regexp.MustCompile("\\d+(st|nd|rd|th)$|\\d{1,2}(/|-)\\d{1,2}(\\d{1,4})$")
}

func lexer(input string) ([]item, error) {
	output := []item{}
	output = append(output, item{itemStart, ""})
	tokens := strings.Split(input, " ")
	idx := 0
	for idx < len(tokens) {
		found := false
		for key, val := range mapper {
			if key == tokens[idx] {
				output = append(output, item{val, tokens[idx]})
				found = true
				break
			}
		}
		if found == false {
			if isCalendar(tokens[idx]) {
				// found a calendar (normalise to / format)
				output = append(output, item{itemCalendar, strings.Replace(tokens[idx], "-", "/", -1)})
			} else {
				timeString, newElement := isTime(tokens[idx])
				if timeString != "" {
					// check for potentialNewString.  If found, add to tokens
					if newElement != "" {
						tokens = append(tokens, "")
						copy(tokens[idx+1:], tokens[idx:])
						tokens[idx+1] = newElement

					}
					output = append(output, item{itemTime, timeString})
				} else if isTimeZone(tokens[idx]) {
					output = append(output, item{itemTimeZone, strings.ToUpper(tokens[idx])})
				} else {
					output = append(output, item{itemText, tokens[idx]})
				}
			}
		}
		idx++
	}
	output = append(output, item{itemEnd, ""})
	return output, nil
}

func normalize(content string) string {
	content = strings.ToLower(content)
	content = strings.TrimSpace(content)
	r, _ := regexp.Compile("\\s+")
	return r.ReplaceAllString(content, " ")
}

func generateMappedRegex(regex *regexp.Regexp, input string) map[string]string {
	match := regex.FindStringSubmatch(input)
	result := make(map[string]string)
	for i, name := range regex.SubexpNames() {
		result[name] = match[i]
	}
	return result
}

func isTime(potentialTime string) (string, string) {
	// Note: if we find an am/pm modifier we push this on to the stack
	// this ensures the lexer handles "2pm" and "2 pm" consistently
	// because the tokenizer would have already generated a new element for
	// the second case.
	outputStr := ""
	additionalElement := ""
	if RE_TIME.MatchString(potentialTime) {
		regmap := generateMappedRegex(RE_TIME, potentialTime)
		hourAsInt, _ := strconv.Atoi(regmap["hour"])
		// if minute not defined this resolves to 0.
		minuteAsInt, _ := strconv.Atoi(regmap["minute"])
		if hourAsInt >= 0 && hourAsInt <= 23 && minuteAsInt >= 0 && minuteAsInt <= 59 {
			outputStr = fmt.Sprintf("%02d:%02d", hourAsInt, minuteAsInt)
			additionalElement = regmap["mod"]
		}
	}
	return outputStr, additionalElement
}

func isCalendar(potentialCalendar string) bool {
	if RE_CALENDAR.MatchString(potentialCalendar) {
		return true
	}
	return false
}

func isTimeZone(potentialTimeZone string) bool {
	// stolen shamelessly from parseTimeZone in go/src/pkg/time
	potentialTimeZone = strings.ToUpper(potentialTimeZone)
	if len(potentialTimeZone) < 3 {
		return false
	}
	// Special case 1: This is the only zone with a lower-case letter.
	if len(potentialTimeZone) >= 4 && potentialTimeZone[:4] == "ChST" {
		return true
	}
	// Special case 2: GMT may have an hour offset; treat it specially.
	if potentialTimeZone[:3] == "GMT" {
		return true
	}
	// How many upper-case letters are there? Need at least three, at most five.
	var nUpper int
	for nUpper = 0; nUpper < 6; nUpper++ {
		if nUpper >= len(potentialTimeZone) {
			break
		}
		if c := potentialTimeZone[nUpper]; c < 'A' || 'Z' < c {
			break
		}
	}
	switch nUpper {
	case 0, 1, 2, 6:
		return false
	case 5: // Must end in T to match.
		if potentialTimeZone[4] == 'T' {
			return true
		}
	case 4: // Must end in T to match.
		if potentialTimeZone[3] == 'T' {
			return true
		}
	case 3:
		return true
	}
	return false
}
