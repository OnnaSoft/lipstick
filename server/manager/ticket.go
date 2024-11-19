package manager

import "strconv"

type TickerManager struct {
	value int
}

func (tm *TickerManager) generate() string {
	tm.value++

	if tm.value > 99999999 {
		tm.value = 0
	}

	return strconv.Itoa(tm.value)
}
