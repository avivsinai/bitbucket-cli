package cmdutil

import (
	"testing"
	"time"
)

func TestCalculatePollInterval(t *testing.T) {
	t.Parallel()
	baseInterval := 10 * time.Second
	maxInterval := 2 * time.Minute

	tests := []struct {
		name        string
		iteration   int
		expectedMin time.Duration // With jitter, result should be >= this (minus jitter)
		expectedMax time.Duration // With jitter, result should be <= this (plus jitter)
	}{
		{
			name:        "iteration 0 returns base interval",
			iteration:   0,
			expectedMin: 8 * time.Second,  // 10s - 15% jitter - some margin
			expectedMax: 12 * time.Second, // 10s + 15% jitter + some margin
		},
		{
			name:        "iteration 1 applies 1.5x backoff",
			iteration:   1,
			expectedMin: 12 * time.Second, // 15s - 15% jitter - margin
			expectedMax: 18 * time.Second, // 15s + 15% jitter + margin
		},
		{
			name:        "iteration 2 applies 1.5^2 backoff",
			iteration:   2,
			expectedMin: 18 * time.Second, // 22.5s - 15% jitter - margin
			expectedMax: 27 * time.Second, // 22.5s + 15% jitter + margin
		},
		{
			name:        "iteration 5 approaches max interval",
			iteration:   5,
			expectedMin: 60 * time.Second,  // Should be close to max
			expectedMax: 140 * time.Second, // 120s + jitter + margin
		},
		{
			name:        "iteration 10 caps at max interval",
			iteration:   10,
			expectedMin: 100 * time.Second, // 120s - 15% jitter - margin
			expectedMax: 140 * time.Second, // 120s + 15% jitter + margin
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Run multiple times to account for jitter randomness
			for i := 0; i < 10; i++ {
				got := PollInterval(baseInterval, maxInterval, tt.iteration)
				if got < tt.expectedMin {
					t.Errorf("PollInterval() = %v, want >= %v", got, tt.expectedMin)
				}
				if got > tt.expectedMax {
					t.Errorf("PollInterval() = %v, want <= %v", got, tt.expectedMax)
				}
			}
		})
	}
}

func TestCalculatePollIntervalCapsAtMax(t *testing.T) {
	t.Parallel()
	baseInterval := 10 * time.Second
	maxInterval := 30 * time.Second

	// After enough iterations, should cap at max (with jitter)
	for iteration := 10; iteration <= 20; iteration++ {
		got := PollInterval(baseInterval, maxInterval, iteration)
		// With 15% jitter, max should be ~34.5s
		if got > 35*time.Second {
			t.Errorf("iteration %d: PollInterval() = %v, should cap near %v", iteration, got, maxInterval)
		}
	}
}

func TestAddJitter(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		duration time.Duration
	}{
		{"10 seconds", 10 * time.Second},
		{"1 minute", 1 * time.Minute},
		{"2 minutes", 2 * time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Run multiple times to verify jitter is applied
			results := make(map[time.Duration]bool)
			for i := 0; i < 100; i++ {
				got := addPollJitter(tt.duration)
				results[got] = true

				// Verify within expected bounds (±15% + 1s margin)
				minExpected := time.Duration(float64(tt.duration) * 0.84) // 1 - 0.15 - small margin
				maxExpected := time.Duration(float64(tt.duration) * 1.16) // 1 + 0.15 + small margin

				if got < minExpected {
					t.Errorf("addPollJitter(%v) = %v, want >= %v", tt.duration, got, minExpected)
				}
				if got > maxExpected {
					t.Errorf("addPollJitter(%v) = %v, want <= %v", tt.duration, got, maxExpected)
				}
			}

			// Verify we got some variation (jitter is working)
			if len(results) < 5 {
				t.Errorf("addPollJitter() produced only %d unique values in 100 runs, expected more variation", len(results))
			}
		})
	}
}

func TestAddJitterMinimum(t *testing.T) {
	t.Parallel()
	// Very small durations should not go below 1 second
	got := addPollJitter(500 * time.Millisecond)
	if got < time.Second {
		t.Errorf("addPollJitter(500ms) = %v, want >= 1s minimum", got)
	}
}

func TestAddJitterZeroAndNegative(t *testing.T) {
	t.Parallel()
	// Zero duration should return zero
	if got := addPollJitter(0); got != 0 {
		t.Errorf("addPollJitter(0) = %v, want 0", got)
	}

	// Negative duration should return unchanged
	neg := -5 * time.Second
	if got := addPollJitter(neg); got != neg {
		t.Errorf("addPollJitter(%v) = %v, want %v", neg, got, neg)
	}
}

func TestBackoffProgression(t *testing.T) {
	t.Parallel()
	// Verify the backoff progression is monotonically increasing (before hitting cap)
	baseInterval := 10 * time.Second
	maxInterval := 5 * time.Minute

	// Calculate expected values without jitter
	expectedBase := []float64{10, 15, 22.5, 33.75, 50.625, 75.9375, 113.90625, 170.859375}

	for i := 0; i < len(expectedBase)-1; i++ {
		// Run multiple times and take average to smooth out jitter
		var sum1, sum2 time.Duration
		runs := 20
		for j := 0; j < runs; j++ {
			sum1 += PollInterval(baseInterval, maxInterval, i)
			sum2 += PollInterval(baseInterval, maxInterval, i+1)
		}
		avg1 := sum1 / time.Duration(runs)
		avg2 := sum2 / time.Duration(runs)

		// Each iteration should be roughly 1.5x the previous (with tolerance for jitter)
		ratio := float64(avg2) / float64(avg1)
		if ratio < 1.3 || ratio > 1.7 {
			t.Errorf("backoff ratio between iteration %d and %d: got %.2f, want ~1.5", i, i+1, ratio)
		}
	}
}
