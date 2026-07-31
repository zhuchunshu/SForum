package adminoverview

import (
	"math"

	"github.com/shirou/gopsutil/v4/load"
)

func sampleSystemLoad() (SystemLoadAverage, bool) {
	average, err := load.Avg()
	if err != nil || average == nil {
		return SystemLoadAverage{}, false
	}

	result := SystemLoadAverage{
		OneMinute:      average.Load1,
		FiveMinutes:    average.Load5,
		FifteenMinutes: average.Load15,
	}
	if !validLoadAverage(result) {
		return SystemLoadAverage{}, false
	}
	return result, true
}

func validLoadAverage(average SystemLoadAverage) bool {
	for _, value := range []float64{average.OneMinute, average.FiveMinutes, average.FifteenMinutes} {
		if value < 0 || math.IsNaN(value) || math.IsInf(value, 0) {
			return false
		}
	}
	return true
}
