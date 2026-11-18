package main

import (
	"testing"
	"time"
)

// TestSlow sleeps past the budget of a second, to see the gate refuse it.
func TestSlow(t *testing.T) { time.Sleep(1100 * time.Millisecond) }
