package types

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

func Parse(input string) (Window, error) {
	input = normalize(input)
	items, err := lexer(input)
	if err != nil {
		return Window{}, err
	}
	p := &parser{
		items:  items,
		window: Window{},
	}
	p.run()
	if p.window.Error != "" {
		return Window{}, errors.New(p.window.Error)
	} else {
		return p.window, nil
	}
}

func (p *parser) run() {
	for state := parseStart; state != nil; {
		state = state(p)
	}
}

// DAG

func parseStart(p *parser) stateFn {
	return switchOnValidStates(p, []itemType{itemTime, itemAnyAlways, itemNever})
}

func parseTime(p *parser) stateFn {
	second := false
	if p.window.Start == "" {
		p.window.Start = p.items[p.pos].val
	} else {
		second = true
		p.window.End = p.items[p.pos].val
	}
	if second {
		return switchOnValidStates(p, []itemType{itemTimeModifier, itemRecurrence, itemOn})
	} else {
		return switchOnValidStates(p, []itemType{itemTimeModifier, itemTimeRange})
	}
}

func parseAny(p *parser) stateFn {
	p.window.AlwaysOn = true
	return switchOnValidStates(p, []itemType{itemAnyAlwaysClarification, itemException, itemEnd})
}

func parseAnyAlwaysClarification(p *parser) stateFn {
	return switchOnValidStates(p, []itemType{itemException, itemEnd})
}

func parseNever(p *parser) stateFn {
	p.window.AlwaysOff = true
	return switchOnValidStates(p, []itemType{itemEnd})
}

func parseException(p *parser) stateFn {
	return switchOnValidStates(p, []itemType{itemTime, itemCalendar})
}

func parseRange(p *parser) stateFn {
	return switchOnValidStates(p, []itemType{itemTime})
}

func parseTimeModifier(p *parser) stateFn {
	timeToUpdateStr := p.window.Start
	if p.window.End != "" {
		timeToUpdateStr = p.window.End
	}
	elements := strings.Split(timeToUpdateStr, ":")
	timeToUpdate, err := strconv.Atoi(elements[0])
	if err == nil {
		if p.items[p.pos].val == "am" {
			if timeToUpdate >= 12 {
				// 12 am is actually 00:00, not 12 noon (as per wikipedia)
				timeToUpdate = timeToUpdate - 12
			}
		} else {
			if timeToUpdate >= 0 && timeToUpdate < 12 {
				timeToUpdate = timeToUpdate + 12
			}
		}
		if p.window.End == "" {
			p.window.Start = fmt.Sprintf("%02d:%s", timeToUpdate, elements[1])
		} else {
			p.window.End = fmt.Sprintf("%02d:%s", timeToUpdate, elements[1])
		}
	}
	return switchOnValidStates(p, []itemType{itemTimeRange, itemOn, itemRecurrence})
}

func parseRecurrence(p *parser) stateFn {
	return switchOnValidStates(p, []itemType{itemDay, itemCalendar, itemRecurringByDayMonthYear})
}

func parseDayMonth(p *parser) stateFn {
	p.updateRecurrence(p.items[p.pos].val)
	return switchOnValidStates(p, []itemType{itemWhere, itemEnd})
}

func parseCalendar(p *parser) stateFn {
	elements := strings.Split(p.items[p.pos].val, "/")
	if len(elements) == 3 {
		if len(elements[2]) == 2 {
			p.window.OnDate = elements[0] + "/" + elements[1] + "/20" + elements[2]
		} else {
			p.window.OnDate = p.items[p.pos].val
		}
		return switchOnValidStates(p, []itemType{itemWhere, itemEnd})
	} else {
		newStr := strings.Replace(p.items[p.pos].val, "st", "", -1)
		newStr = strings.Replace(newStr, "nd", "", -1)
		newStr = strings.Replace(newStr, "rd", "", -1)
		newStr = strings.Replace(newStr, "th", "", -1)
		p.updateRecurrence(newStr)
		return switchOnValidStates(p, []itemType{itemRecurringByDayMonthYear, itemMonth, itemClarification})
	}
}

func parseClarification(p *parser) stateFn {
	return switchOnValidStates(p, []itemType{itemRecurringByDayMonthYear, itemMonth, itemClarification})
}

func parseRecurringByDayMonthYear(p *parser) stateFn {
	item_ := p.getLastElement(itemCalendar)
	elements := strings.Split(item_.val, "/")
	if p.items[p.pos].val == "year" || p.items[p.pos].val == "yearly" {
		if len(elements) != 2 {
			// not valid.  not properly qualified, e.g. "1st yearly"
			p.window.Error = fmt.Sprintf("Not a valid recurrence for: \"%s\"", item_.val)
			return nil
		}
	}
	p.updateRecurrence(p.items[p.pos].val)
	return switchOnValidStates(p, []itemType{itemWhere, itemEnd})
}

func parseOn(p *parser) stateFn {
	return switchOnValidStates(p, []itemType{itemCalendar})
}

func parseWhere(p *parser) stateFn {
	return switchOnValidStates(p, []itemType{itemTimeZoneParam})
}

func parseTimeZoneParam(p *parser) stateFn {
	return switchOnValidStates(p, []itemType{itemEquals})
}

func parseEquals(p *parser) stateFn {
	return switchOnValidStates(p, []itemType{itemTimeZone})
}

func parseTimeZone(p *parser) stateFn {
	p.window.Timezone = p.items[p.pos].val
	return switchOnValidStates(p, []itemType{itemEnd})
}

func parseEnd(p *parser) stateFn {
	return nil
}

// Utilities

func (p *parser) next() item {
	if p.pos >= len(p.items) {
		return item{itemEnd, ""}
	}
	p.pos++
	return p.items[p.pos]
}

func (p *parser) getLastElement(typ itemType) item {
	for i := p.pos; i >= 0; i-- {
		item_ := p.items[i]
		if item_.typ == typ {
			return item_
		}
	}
	// we shouldnt reach this position (typically) because the DAG should ensure we are able to find a valid item
	return item{}
}

func (p *parser) updateRecurrence(item string) {
	if p.window.Recurrence == "" {
		p.window.Recurrence += item
	} else {
		p.window.Recurrence += " " + item
	}
}

func switchOnValidStates(p *parser, validStates []itemType) stateFn {
	n := p.next()
	// check if n.typ is in validStates. If so switch and return, else set error and return nil
	valid := false
	for _, element := range validStates {
		if n.typ == element {
			valid = true
		}
	}
	if valid == true {
		switch {
		case n.typ == itemEnd:
			return parseEnd
		case n.typ == itemTime:
			return parseTime
		case n.typ == itemTimeRange:
			return parseRange
		case n.typ == itemTimeModifier:
			return parseTimeModifier
		case n.typ == itemWhere:
			return parseWhere
		case n.typ == itemEquals:
			return parseEquals
		case n.typ == itemRecurrence:
			return parseRecurrence
		case n.typ == itemAnyAlways:
			return parseAny
		case n.typ == itemAnyAlwaysClarification:
			return parseAnyAlwaysClarification
		case n.typ == itemNever:
			return parseNever
		case n.typ == itemException:
			return parseException
		case n.typ == itemOn:
			return parseOn
		case n.typ == itemTimeZoneParam:
			return parseTimeZoneParam
		case n.typ == itemTimeZone:
			return parseTimeZone
		case n.typ == itemCalendar:
			return parseCalendar
		case n.typ == itemDay:
			return parseDayMonth
		case n.typ == itemMonth:
			return parseDayMonth
		case n.typ == itemRecurringByDayMonthYear:
			return parseRecurringByDayMonthYear
		case n.typ == itemClarification:
			return parseClarification
		}
	}
	p.window.Error = fmt.Sprintf("Invalid parse at: \"%s\"", n.val)
	return nil
}
