package utils

import (
	"fmt"
	"sync"
	"time"
)

type PerformanceTracker struct {
	metrics map[string][]time.Duration
	mu      sync.Mutex
}

func NewPerformanceTracker() *PerformanceTracker {
	return &PerformanceTracker{
		metrics: make(map[string][]time.Duration),
	}
}

func (pt *PerformanceTracker) TrackOperation(operation string, duration time.Duration) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if pt.metrics == nil {
		pt.metrics = make(map[string][]time.Duration)
	}
	pt.metrics[operation] = append(pt.metrics[operation], duration)
}

func (pt *PerformanceTracker) GenerateAggregateReport() string {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	var report string
	report = "Performance Report:\n"

	for op, durations := range pt.metrics {
		var total time.Duration
		for _, d := range durations {
			total += d
		}
		avg := total / time.Duration(len(durations))

		report += fmt.Sprintf("%s:\n", op)
		report += fmt.Sprintf("  Count: %d\n", len(durations))
		report += fmt.Sprintf("  Average: %v\n", avg)
		report += fmt.Sprintf("  Total: %v\n", total)
	}

	return report
}
