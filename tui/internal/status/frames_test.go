package status

import "time"

// The dot's frames take no time in the tests, which step it through every frame without watching it move.
func init() { frameEvery = time.Microsecond }
