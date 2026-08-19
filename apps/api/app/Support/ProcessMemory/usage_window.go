package processmemory

import (
	"sort"
	"sync"
	"time"
)

type usagePoint struct {
	at            time.Time
	api           uint64
	worker        uint64
	plugin        uint64
	total         uint64
	pssAvailable  bool
	apiPSS        uint64
	workerPSS     uint64
	pluginPSS     uint64
	totalPSS      uint64
	pluginPSSByID map[string]uint64
}

// UsageWindow 为同一采样器维护固定时窗中位数；相同 sampledAt 的缓存帧只计一次。
type UsageWindow struct {
	mu     sync.Mutex
	window time.Duration
	points []usagePoint
}

func NewUsageWindow(window time.Duration) *UsageWindow {
	if window <= 0 {
		window = DefaultUsageWindow
	}
	return &UsageWindow{window: window}
}

func (w *UsageWindow) Observe(usage RuntimeUsage) RuntimeUsage {
	if w == nil {
		return usage
	}
	at := time.Now().UTC()
	if usage.SampledAt != nil && !usage.SampledAt.IsZero() {
		at = usage.SampledAt.UTC()
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.points) == 0 || !w.points[len(w.points)-1].at.Equal(at) {
		pluginPSSByID := map[string]uint64{}
		for _, plugin := range usage.Plugins {
			if plugin.PSSBytes > 0 {
				pluginPSSByID[plugin.ExtensionID] = plugin.PSSBytes
			}
		}
		w.points = append(w.points, usagePoint{
			at: at, api: usage.APIMemoryBytes, worker: usage.WorkerMemoryBytes,
			plugin: usage.PluginMemoryBytes, total: usage.TotalMemoryBytes,
			pssAvailable: usage.TotalPSSBytes > 0,
			apiPSS:       usage.APIPSSBytes, workerPSS: usage.WorkerPSSBytes,
			pluginPSS: usage.PluginPSSBytes, totalPSS: usage.TotalPSSBytes,
			pluginPSSByID: pluginPSSByID,
		})
	}
	cutoff := at.Add(-w.window)
	first := 0
	for first < len(w.points) && !w.points[first].at.After(cutoff) {
		first++
	}
	if first > 0 {
		w.points = append([]usagePoint(nil), w.points[first:]...)
	}
	usage.MemorySampleCount = len(w.points)
	usage.MemoryWindowSeconds = max(int(w.window.Seconds()), 0)
	usage.APIMemoryMedianBytes = medianUsageValue(w.points, func(point usagePoint) uint64 { return point.api })
	usage.WorkerMemoryMedianBytes = medianUsageValue(w.points, func(point usagePoint) uint64 { return point.worker })
	usage.PluginMemoryMedianBytes = medianUsageValue(w.points, func(point usagePoint) uint64 { return point.plugin })
	usage.TotalMemoryMedianBytes = medianUsageValue(w.points, func(point usagePoint) uint64 { return point.total })
	pssPoints := make([]usagePoint, 0, len(w.points))
	for _, point := range w.points {
		if point.pssAvailable {
			pssPoints = append(pssPoints, point)
		}
	}
	usage.PSSSampleCount = len(pssPoints)
	usage.APIPSSMedianBytes = medianUsageValue(pssPoints, func(point usagePoint) uint64 { return point.apiPSS })
	usage.WorkerPSSMedianBytes = medianUsageValue(pssPoints, func(point usagePoint) uint64 { return point.workerPSS })
	usage.PluginPSSMedianBytes = medianUsageValue(pssPoints, func(point usagePoint) uint64 { return point.pluginPSS })
	usage.TotalPSSMedianBytes = medianUsageValue(pssPoints, func(point usagePoint) uint64 { return point.totalPSS })
	for index := range usage.Plugins {
		values := make([]uint64, 0, len(pssPoints))
		for _, point := range pssPoints {
			if value, ok := point.pluginPSSByID[usage.Plugins[index].ExtensionID]; ok {
				values = append(values, value)
			}
		}
		usage.Plugins[index].PSSSampleCount = len(values)
		usage.Plugins[index].PSSMedianBytes = medianUint64(values)
	}
	return usage
}

func medianUsageValue(points []usagePoint, value func(usagePoint) uint64) uint64 {
	if len(points) == 0 {
		return 0
	}
	values := make([]uint64, len(points))
	for index, point := range points {
		values[index] = value(point)
	}
	return medianUint64(values)
}

func medianUint64(values []uint64) uint64 {
	if len(values) == 0 {
		return 0
	}
	values = append([]uint64(nil), values...)
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	middle := len(values) / 2
	if len(values)%2 == 1 {
		return values[middle]
	}
	return values[middle-1] + (values[middle]-values[middle-1])/2
}
