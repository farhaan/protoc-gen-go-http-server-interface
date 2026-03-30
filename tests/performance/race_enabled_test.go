//go:build race

package performance_test

// maxAllocsPerRequest is raised under the race detector because the detector
// itself adds shadow-memory tracking overhead that shows up as extra allocations.
// Set to 30 to accommodate variability across Go versions and platforms.
const maxAllocsPerRequest = 30.0
