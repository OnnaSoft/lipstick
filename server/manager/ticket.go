package manager

import (
	"math"
	"strconv"
)

type TickerManager struct {
	value uint64
}

func (tm *TickerManager) generate() string {
	tm.value++

	if tm.value > math.MaxUint64 {
		tm.value = 0
	}

	return strconv.FormatUint(tm.value, 10)
}
