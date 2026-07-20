package cmdutil

import (
	"crypto/rand"
	"math/big"
	"time"
)

// pollBackoffMultiplier is the factor by which the polling interval increases each iteration
const pollBackoffMultiplier = 1.5

// pollJitterFraction is the maximum random adjustment (±15%) applied to intervals
const pollJitterFraction = 0.15

// PollInterval computes the next polling interval using exponential backoff with jitter.
// The formula is: min(baseInterval * multiplier^iteration, maxInterval) ± jitter
func PollInterval(baseInterval, maxInterval time.Duration, iteration int) time.Duration {
	if iteration <= 0 {
		return addPollJitter(baseInterval)
	}

	// Calculate exponential backoff: base * 1.5^iteration
	interval := float64(baseInterval)
	for i := 0; i < iteration; i++ {
		interval *= pollBackoffMultiplier
		if interval >= float64(maxInterval) {
			interval = float64(maxInterval)
			break
		}
	}

	// Cap at max interval
	if interval > float64(maxInterval) {
		interval = float64(maxInterval)
	}

	return addPollJitter(time.Duration(interval))
}

// addPollJitter applies ±15% random jitter to a duration to prevent thundering herd.
// Uses crypto/rand for better randomness distribution.
func addPollJitter(d time.Duration) time.Duration {
	if d <= 0 {
		return d
	}

	// Calculate jitter range: ±15% of the duration
	jitterRange := int64(float64(d) * pollJitterFraction * 2) // Total range is 2x the fraction
	if jitterRange <= 0 {
		return d
	}

	// Generate random value in range [0, jitterRange)
	n, err := rand.Int(rand.Reader, big.NewInt(jitterRange))
	if err != nil {
		// Fallback to no jitter on error
		return d
	}

	// Apply jitter: subtract half the range, then add random value
	// This gives us a value in [-pollJitterFraction, +pollJitterFraction]
	jitter := n.Int64() - (jitterRange / 2)
	result := time.Duration(int64(d) + jitter)

	// Ensure we don't go below 1 second minimum
	if result < time.Second {
		result = time.Second
	}

	return result
}
