package parser

import (
	"github.com/kieranbroadfoot/horae/types"
)

type stateFn func(*parser) stateFn

type itemType int

type item struct {
	typ itemType
	val string
}

type parser struct {
	pos    int // current position in the input.
	items  []item
	window types.Window
}

const (
	itemStart itemType = iota
	itemEnd
	itemText
	itemTime
	itemTimeRange
	itemTimeModifier
	itemWhere
	itemEquals
	itemAnd
	itemRecurrence
	itemAnyAlways
	itemAnyAlwaysClarification
	itemException
	itemOn
	itemTimeZoneParam
	itemTimeZone
	itemCalendar
	itemDay
	itemMonth
	itemRecurringByDayMonthYear
	itemClarification
)
