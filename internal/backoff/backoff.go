// Package backoff holds the retry schedule from PROTOCOL.md section 8.
package backoff

import "time"

// Steady is the delay once the fast retries are used up, and the normal periodic interval.
const Steady = 5 * time.Minute

// Delay returns the wait before the next attempt after `failures` consecutive failures:
// 1 → 5s, 2 → 15s, 3 → 60s, 4 or more → Steady. failures <= 0 → Steady (nothing failed,
// so just the periodic check).
func Delay(failures int) time.Duration {
	panic("not implemented")
}
