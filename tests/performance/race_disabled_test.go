//go:build !race

package performance_test

// maxAllocsPerRequest is the baseline without the race detector.
const maxAllocsPerRequest = 10.0
