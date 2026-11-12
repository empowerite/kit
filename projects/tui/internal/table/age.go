// How long ago, and how long something took, as a table draws a time.
package table

import (
	"fmt"
	"strconv"
)

// Age gives a duration in its coarsest informative unit, as kit flow draws one: 45s, 12m, 3h, 47d.
func Age(secs int64) string {
	if secs < 0 {
		secs = 0
	}
	switch {
	case secs < 120:
		return strconv.FormatInt(secs, 10) + "s"
	case secs < 7200:
		return strconv.FormatInt(secs/60, 10) + "m"
	case secs < 172800:
		return strconv.FormatInt(secs/3600, 10) + "h"
	}
	return strconv.FormatInt(secs/86400, 10) + "d"
}

// Took gives how long a build or a job Took, to the second: 45s, 3m 36s, 1h 02m; … while it runs.
func Took(secs *int64) string {
	switch {
	case secs == nil:
		return "…"
	case *secs < 60:
		return strconv.FormatInt(max(*secs, 0), 10) + "s"
	case *secs < 3600:
		return fmt.Sprintf("%dm %02ds", *secs/60, *secs%60)
	}
	return fmt.Sprintf("%dh %02dm", *secs/3600, *secs%3600/60)
}
