//go:build !race

package table

// raceEnabled says the tests run under the race detector, which slows them about tenfold.
const raceEnabled = false
